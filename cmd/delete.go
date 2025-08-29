package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/rodneyxr/mpkube/pkg/multipass"
	"github.com/spf13/cobra"
)

// NewDeleteCmd creates a command to delete a k3s cluster
func NewDeleteCmd() *cobra.Command {
	var force bool

	deleteCmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete a k3s cluster",
		Long:  `Delete a Kubernetes cluster by removing the underlying Multipass VM.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			return deleteCluster(name, force)
		},
	}

	// Add flags
	deleteCmd.Flags().BoolVarP(&force, "force", "f", false, "Force deletion without confirmation")

	return deleteCmd
}

// deleteCluster deletes a k3s cluster by removing the Multipass VM
func deleteCluster(name string, force bool) error {
	mp, err := multipass.NewMultipassEnv()
	if err != nil {
		return fmt.Errorf("failed to initialize multipass environment: %w", err)
	}

	// If name doesn't have mpkube- prefix, add it
	if !strings.HasPrefix(name, "mpkube-") {
		name = fmt.Sprintf("mpkube-%s", name)
	}

	// Check if the VM exists
	vm, err := mp.GetVMByName(name)
	if err != nil {
		return fmt.Errorf("cluster '%s' not found: %w", name, err)
	}

	// Confirmation unless force flag is used
	if !force {
		fmt.Printf("Are you sure you want to delete cluster '%s' (IP: %s)? [y/N]: ", vm.Name, vm.IPv4)
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	fmt.Printf("Deleting cluster '%s'...\n", name)

	// Delete the VM
	if err := mp.DeleteVM(name); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	fmt.Printf("Cluster '%s' deleted successfully.\n", name)
	return nil
}
