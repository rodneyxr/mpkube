package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/rodneyxr/mpkube/pkg/multipass"
	"github.com/spf13/cobra"
)

// NewListCmd creates a command to list all k3s clusters
func NewListCmd() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all k3s clusters",
		Long:  `List all Kubernetes clusters created with this tool in Multipass VMs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listClusters()
		},
	}

	return listCmd
}

// listClusters lists all clusters managed by this tool
func listClusters() error {
	mp, err := multipass.NewMultipassEnv()
	if err != nil {
		return fmt.Errorf("failed to initialize multipass environment: %w", err)
	}

	// Get all VMs that have our cluster prefix
	vms, err := mp.GetK3sVMs()
	if err != nil {
		return fmt.Errorf("failed to list VMs: %w", err)
	}

	if len(vms) == 0 {
		fmt.Println("No K3s clusters found.")
		return nil
	}

	// Print table of clusters
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATE\tIP\tIMAGE")

	for _, vm := range vms {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", vm.Name, vm.State, vm.IPv4, vm.Image)
	}

	w.Flush()
	return nil
}
