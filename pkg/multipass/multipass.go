package multipass

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// MultipassEnv represents the Multipass environment
type MultipassEnv struct {
	IsWSL            bool
	RunningOnWindows bool
	UseWSLMultipass  bool
	MultipassCmd     string
	WSLDistro        string
}

// NewMultipassEnv initializes a new MultipassEnv
func NewMultipassEnv() (*MultipassEnv, error) {
	m := &MultipassEnv{
		RunningOnWindows: runtime.GOOS == "windows",
	}

	// Check if we're running in WSL
	m.IsWSL = isWSL()

	// Determine multipass command
	cmd, useWSLMultipass, wslDistro, err := getMultipassCmd(m.IsWSL, m.RunningOnWindows)
	if err != nil {
		return nil, err
	}

	m.MultipassCmd = cmd
	m.UseWSLMultipass = useWSLMultipass
	m.WSLDistro = wslDistro
	return m, nil
}

// isWSL checks if we're running in Windows Subsystem for Linux
func isWSL() bool {
	// If we're on Windows, we're not in WSL
	if runtime.GOOS == "windows" {
		return false
	}

	_, err := os.Stat("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}

	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}

	return bytes.Contains(data, []byte("WSL"))
}

// getMultipassCmd returns the appropriate multipass command based on the environment
func getMultipassCmd(isWSL bool, isWindows bool) (string, bool, string, error) {
	// Running on Windows but not in WSL
	if isWindows {
		// Check for native Windows Multipass first
		winPaths := []string{
			"C:\\Program Files\\Multipass\\bin\\multipass.exe",
			"multipass.exe", // If it's in PATH
		}

		for _, path := range winPaths {
			if _, err := exec.LookPath(path); err == nil {
				return path, false, "", nil
			}
		}

		// If not found, check if we can access multipass through WSL
		wslDistro, wslAvailable := checkWSLAvailable()
		if wslAvailable {
			// Try to verify multipass exists in the WSL environment
			// Use --shell-type login to ensure the environment is properly loaded
			cmd := exec.Command("wsl", "-d", wslDistro, "--shell-type", "login", "which", "multipass")
			if err := cmd.Run(); err == nil {
				// Multipass exists in WSL
				return "multipass", true, wslDistro, nil
			}
		}

		return "", false, "", fmt.Errorf("multipass not found in Windows or WSL")
	}

	// Running in WSL
	if isWSL {
		paths := []string{
			"/mnt/c/Program Files/Multipass/bin/multipass.exe",
			"/mnt/c/Windows/System32/multipass.exe",
		}

		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				return path, false, "", nil
			}
		}

		// Try native multipass in WSL
		if _, err := exec.LookPath("multipass"); err == nil {
			return "multipass", false, "", nil
		}

		return "", false, "", fmt.Errorf("multipass not found in WSL or Windows path")
	}

	// Not in WSL, just check if multipass is available
	if _, err := exec.LookPath("multipass"); err == nil {
		return "multipass", false, "", nil
	}

	return "", false, "", fmt.Errorf("multipass command not found")
}

// checkWSLAvailable checks if WSL is available and returns the default distribution
func checkWSLAvailable() (string, bool) {
	// Check if WSL command exists
	if _, err := exec.LookPath("wsl"); err != nil {
		return "", false
	}

	// Get default WSL distribution list (raw output)
	cmd := exec.Command("wsl", "-l", "-q")
	outputBytes, err := cmd.Output()
	if err != nil {
		// If listing fails, WSL might still be available but without distributions or with an older version
		// We can try a simpler check or assume it's not usable for our purpose here.
		// For now, let's return false if listing fails.
		return "", false
	}

	// Decode UTF-16LE output from wsl -l -q
	utf16Decoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
	reader := transform.NewReader(bytes.NewReader(outputBytes), utf16Decoder)
	decodedBytes, err := io.ReadAll(reader)
	if err != nil {
		// Fallback or error handling if decoding fails
		// Maybe try assuming UTF-8 as a last resort? Or just fail.
		fmt.Fprintf(os.Stderr, "Warning: Failed to decode WSL distribution list as UTF-16: %v. Trying as UTF-8.\n", err)
		decodedBytes = outputBytes // Use original bytes if decoding fails
	}

	// Parse the distribution list
	outputStr := string(decodedBytes)
	distros := strings.Split(outputStr, "\n")
	if len(distros) == 0 {
		return "", false
	}

	// Find the first valid distribution name
	for _, distro := range distros {
		// Clean up the name: remove carriage returns and trim whitespace
		cleanedDistro := strings.TrimSpace(strings.ReplaceAll(distro, "\r", ""))
		if cleanedDistro != "" {
			// Check if this distribution is actually running or usable
			// A simple check could be trying to run 'true' inside it
			checkCmd := exec.Command("wsl", "-d", cleanedDistro, "true")
			if checkCmd.Run() == nil {
				return cleanedDistro, true
			}
			// If the check fails, continue to the next potential default
			fmt.Fprintf(os.Stderr, "Warning: WSL distribution '%s' found but seems unavailable or stopped. Trying next.\n", cleanedDistro)
		}
	}

	// If no valid/running distribution was found after checking all lines
	return "", false
}

// RunMultipassCmd executes a multipass command and returns the output
func (m *MultipassEnv) RunMultipassCmd(args ...string) (string, error) {
	var cmd *exec.Cmd

	// Windows using WSL multipass
	if m.RunningOnWindows && m.UseWSLMultipass {
		// Use --shell-type login to ensure the environment is properly loaded
		wslArgs := []string{"-d", m.WSLDistro, "--shell-type", "login", "multipass"}
		wslArgs = append(wslArgs, args...)
		cmd = exec.Command("wsl", wslArgs...)
	} else if m.IsWSL && strings.HasSuffix(m.MultipassCmd, ".exe") {
		// WSL using Windows multipass.exe
		wslArgs := []string{"/c", m.MultipassCmd}
		wslArgs = append(wslArgs, args...)
		cmd = exec.Command("cmd.exe", wslArgs...)
	} else {
		// Native multipass in current environment
		cmd = exec.Command(m.MultipassCmd, args...)
	}

	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ListVMs returns a list of multipass VMs
func (m *MultipassEnv) ListVMs() ([]VM, error) {
	output, err := m.RunMultipassCmd("list", "--format", "csv")
	if err != nil {
		return nil, err
	}

	return parseMultipassList(output)
}

// VM represents a multipass virtual machine
type VM struct {
	Name  string
	State string
	IPv4  string
	Image string
	IsK3s bool
}

// parseMultipassList parses the output of multipass list command
func parseMultipassList(output string) ([]VM, error) {
	var vms []VM

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) <= 1 {
		return vms, nil // No VMs or just header
	}

	// Skip the header line
	for _, line := range lines[1:] {
		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}

		vm := VM{
			Name:  strings.TrimSpace(parts[0]),
			State: strings.TrimSpace(parts[1]),
			IPv4:  strings.TrimSpace(parts[2]),
			Image: strings.TrimSpace(parts[4]),
		}

		// Check if this is a K3s VM by looking for cluster prefix
		if strings.HasPrefix(vm.Name, "mpkube-") {
			vm.IsK3s = true
		}

		vms = append(vms, vm)
	}

	return vms, nil
}

// GetVMByName returns a VM by name
func (m *MultipassEnv) GetVMByName(name string) (*VM, error) {
	vms, err := m.ListVMs()
	if err != nil {
		return nil, err
	}

	for _, vm := range vms {
		if vm.Name == name {
			return &vm, nil
		}
	}

	return nil, fmt.Errorf("VM %s not found", name)
}

// GetK3sVMs returns all K3s VMs
func (m *MultipassEnv) GetK3sVMs() ([]VM, error) {
	vms, err := m.ListVMs()
	if err != nil {
		return nil, err
	}

	var k3sVMs []VM
	for _, vm := range vms {
		if strings.HasPrefix(vm.Name, "mpkube-") {
			k3sVMs = append(k3sVMs, vm)
		}
	}

	return k3sVMs, nil
}

// DeleteVM deletes and purges a multipass VM by name
func (m *MultipassEnv) DeleteVM(name string) error {
	// First, stop the VM if it's running. Ignore errors if it's already stopped or doesn't exist.
	_, _ = m.RunMultipassCmd("stop", name)

	// Delete and purge the VM
	output, err := m.RunMultipassCmd("delete", name, "--purge")
	if err != nil {
		// Check if the error indicates the VM was already deleted or not found
		if strings.Contains(output, "does not exist") {
			return nil // Consider it successfully deleted if it doesn't exist
		}
		return fmt.Errorf("failed to delete VM %s: %v\nOutput: %s", name, err, output)
	}
	fmt.Printf("VM %s deleted successfully.\n", name)
	return nil
}
