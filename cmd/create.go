package cmd

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rodneyxr/mpkube/pkg/k3s"
	"github.com/rodneyxr/mpkube/pkg/multipass"
	"github.com/spf13/cobra"
)

// NewCreateCmd creates a command to create a new k3s cluster
func NewCreateCmd() *cobra.Command {
	var cpus int
	var memory string
	var disk string
	var name string

	createCmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new k3s cluster",
		Long:  `Create a new Kubernetes cluster using k3s in a Multipass VM with traefik disabled.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				name = args[0]
			}

			return createCluster(name, cpus, memory, disk)
		},
	}

	// Add flags for customizing the VM
	createCmd.Flags().IntVarP(&cpus, "cpus", "c", 2, "Number of CPUs for the VM")
	createCmd.Flags().StringVarP(&memory, "memory", "m", "2G", "Memory allocation for the VM")
	createCmd.Flags().StringVarP(&disk, "disk", "d", "10G", "Disk space for the VM")
	createCmd.Flags().StringVar(&name, "name", "", "Name for the cluster (defaults to mpkube-<random> or mpkube-default if first cluster)")

	return createCmd
}

// createCluster creates a new k3s cluster in a Multipass VM
func createCluster(name string, cpus int, memory string, disk string) error {
	mp, err := multipass.NewMultipassEnv()
	if err != nil {
		return fmt.Errorf("failed to initialize multipass environment: %w", err)
	}

	// Generate cluster name if not provided
	if name == "" {
		// Check if this is the first cluster
		vms, err := mp.GetK3sVMs()
		if err != nil {
			return fmt.Errorf("failed to list VMs: %w", err)
		}

		if len(vms) == 0 {
			name = "mpkube-default"
		} else {
			// Generate random suffix (similar to k8s pod naming)
			shortID := strings.Split(uuid.New().String(), "-")[0]
			name = fmt.Sprintf("mpkube-%s", shortID)
		}
	}

	// If name doesn't have mpkube- prefix, add it
	if !strings.HasPrefix(name, "mpkube-") {
		name = fmt.Sprintf("mpkube-%s", name)
	}

	fmt.Printf("Creating k3s cluster with name: %s\n", name)

	// Launch the VM
	launchArgs := []string{
		"launch",
		"--name", name,
		"--cpus", fmt.Sprintf("%d", cpus),
		"--memory", memory,
		"--disk", disk,
	}

	// For simplicity, use ubuntu 24.04 LTS
	launchArgs = append(launchArgs, "24.04")

	fmt.Println("Launching Multipass VM...")
	output, err := mp.RunMultipassCmd(launchArgs...)
	if err != nil {
		return fmt.Errorf("failed to launch VM: %w\n%s", err, output)
	}

	// Get the VM's IP address
	vm, err := mp.GetVMByName(name)
	if err != nil {
		return fmt.Errorf("failed to get VM details: %w", err)
	}

	fmt.Printf("VM launched with IP: %s\n", vm.IPv4)
	fmt.Println("Installing k3s (this may take a few minutes)...")

	// Install k3s on the VM
	if err := k3s.InstallK3s(mp, name); err != nil {
		return fmt.Errorf("failed to install k3s: %w", err)
	}

	fmt.Println("K3s installed successfully!")

	fmt.Println("\nCluster created successfully!")
	fmt.Printf("Cluster name: %s\n", name)
	fmt.Printf("Cluster IP: %s\n", vm.IPv4)
	fmt.Println("\nUse the following commands to access the cluster:")
	fmt.Printf("export KUBECONFIG=~/.kube/mpkube/kubeconfig-%s\n", name)
	fmt.Printf("mpkube kubeconfig get %s -o $KUBECONFIG\n", name)

	return nil
}
