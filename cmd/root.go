package cmd

import (
	"github.com/spf13/cobra"
)

// Version is the CLI version
const Version = "0.1.0"

// NewRootCmd creates the root command for the CLI
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "mpkube",
		Short:   "A CLI tool for managing Kubernetes clusters within Multipass",
		Long:    `mpkube is a command line tool for creating and managing Kubernetes clusters, specifically k3s clusters, within Multipass VMs.`,
		Version: Version,
	}

	// Add subcommands
	rootCmd.AddCommand(
		NewListCmd(),
		NewCreateCmd(),
		NewKubeconfigCmd(),
		NewDeleteCmd(),
	)

	return rootCmd
}
