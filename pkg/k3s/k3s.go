package k3s

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rodneyxr/mpkube/pkg/multipass"
)

// InstallK3s installs K3s on a multipass VM without traefik
func InstallK3s(mp *multipass.MultipassEnv, vmName string) error {
	vm, err := mp.GetVMByName(vmName)
	if err != nil {
		return err
	}

	// Prepare the K3s install command with traefik disabled and advertise the VM's IP
	k3sInstallCmd := fmt.Sprintf(
		"curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC=\"--disable=traefik --advertise-address=%s --node-ip=%s\" sh -",
		vm.IPv4, vm.IPv4,
	)

	// Execute the command through multipass, which will handle WSL/Windows integration
	_, err = mp.RunMultipassCmd("exec", vmName, "--", "bash", "-c", k3sInstallCmd)
	return err
}

// GetKubeconfig retrieves kubeconfig from a K3s node
func GetKubeconfig(mp *multipass.MultipassEnv, vmName string) (string, error) {
	output, err := mp.RunMultipassCmd("exec", vmName, "--", "sudo", "cat", "/etc/rancher/k3s/k3s.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Replace localhost with the VM's IP address
	vm, err := mp.GetVMByName(vmName)
	if err != nil {
		return "", err
	}

	kubeconfig := strings.ReplaceAll(output, "127.0.0.1", vm.IPv4)
	kubeconfig = strings.ReplaceAll(kubeconfig, "localhost", vm.IPv4)

	// Set the cluster and context names to match the VM name
	kubeconfig = strings.ReplaceAll(kubeconfig, "default", vmName)

	return kubeconfig, nil
}

// SaveKubeconfig saves the kubeconfig to a file
func SaveKubeconfig(kubeconfig string, outputPath string) error {
	// Handle Windows path conversion if necessary
	outputPath = normalizePath(outputPath)

	if filepath.Ext(outputPath) == "" {
		outputPath = filepath.Join(outputPath, "config")
	}

	// Ensure the directory exists
	dir := filepath.Dir(outputPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Write the kubeconfig to the file
	if err := os.WriteFile(outputPath, []byte(kubeconfig), 0644); err != nil {
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	return nil
}

// MergeKubeconfigs combines multiple kubeconfigs into one
func MergeKubeconfigs(kubeconfigs []string) (string, error) {
	// TODO: Implement merging logic for multiple kubeconfigs
	// For the MVP, we'll just return the first one
	if len(kubeconfigs) == 0 {
		return "", fmt.Errorf("no kubeconfigs provided")
	}

	return kubeconfigs[0], nil
}

// normalizePath handles path conversion between Windows and WSL paths
func normalizePath(path string) string {
	// If on Windows, ensure proper path separators
	return filepath.FromSlash(path)
}
