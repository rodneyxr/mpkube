package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rodneyxr/mpkube/pkg/k3s"
	"github.com/rodneyxr/mpkube/pkg/multipass"
	"github.com/spf13/cobra"
)

// NewKubeconfigCmd creates a command to manage kubeconfigs
func NewKubeconfigCmd() *cobra.Command {
	kubeconfigCmd := &cobra.Command{
		Use:   "kubeconfig",
		Short: "Manage kubeconfig for clusters",
		Long:  `Generate, extract or merge kubeconfig files for k3s clusters created with this tool.`,
	}

	// Add subcommands
	kubeconfigCmd.AddCommand(NewKubeconfigGetCmd())
	kubeconfigCmd.AddCommand(NewKubeconfigMergeCmd())

	return kubeconfigCmd
}

// NewKubeconfigGetCmd creates a command to get kubeconfig for a specific cluster
func NewKubeconfigGetCmd() *cobra.Command {
	var outputFile string

	getCmd := &cobra.Command{
		Use:   "get [mpkube-name]",
		Short: "Get kubeconfig for a specific cluster",
		Long:  `Extract kubeconfig from a specific k3s cluster and print it or save it to a file.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var clusterName string
			if len(args) > 0 {
				clusterName = args[0]
			}
			return getKubeconfig(clusterName, outputFile)
		},
	}

	getCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file to save kubeconfig (prints to stdout if not specified)")

	return getCmd
}

// NewKubeconfigMergeCmd creates a command to merge kubeconfigs from all clusters
func NewKubeconfigMergeCmd() *cobra.Command {
	var outputFile string

	mergeCmd := &cobra.Command{
		Use:   "merge",
		Short: "Merge kubeconfigs from all clusters",
		Long:  `Merge kubeconfigs from all k3s clusters created with this tool into a single config.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mergeKubeconfigs(outputFile)
		},
	}

	mergeCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write kubeconfig to this file (prints to stdout if not specified)")

	return mergeCmd
}

// getKubeconfig retrieves kubeconfig for a specific cluster
func getKubeconfig(clusterName string, outputFile string) error {
	mp, err := multipass.NewMultipassEnv()
	if err != nil {
		return fmt.Errorf("failed to initialize multipass environment: %w", err)
	}

	// If no cluster name provided, list available clusters
	if clusterName == "" {
		vms, err := mp.GetK3sVMs()
		if err != nil {
			return fmt.Errorf("failed to list clusters: %w", err)
		}

		if len(vms) == 0 {
			return fmt.Errorf("no clusters found")
		} else if len(vms) == 1 {
			// If there's only one cluster, use it
			clusterName = vms[0].Name
			fmt.Printf("Using cluster: %s\n", clusterName)
		} else {
			fmt.Println("Please specify one of the available clusters:")
			for _, vm := range vms {
				fmt.Printf("  %s\n", vm.Name)
			}
			return fmt.Errorf("cluster name required")
		}
	}

	// Add mpkube- prefix if not present
	if !strings.HasPrefix(clusterName, "mpkube-") {
		clusterName = fmt.Sprintf("mpkube-%s", clusterName)
	}

	// Get kubeconfig from the specified cluster
	kubeconfig, err := k3s.GetKubeconfig(mp, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Save or print the kubeconfig
	if outputFile != "" {
		// Ensure directory exists
		dir := filepath.Dir(outputFile)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		}

		// Write kubeconfig to file
		if err := os.WriteFile(outputFile, []byte(kubeconfig), 0644); err != nil {
			return fmt.Errorf("failed to write kubeconfig: %w", err)
		}

		fmt.Printf("Kubeconfig saved to: %s\n", outputFile)
	} else {
		// Print to stdout
		fmt.Println(kubeconfig)
	}

	return nil
}

// mergeKubeconfigs merges kubeconfigs from all clusters
func mergeKubeconfigs(outputFile string) error {
	mp, err := multipass.NewMultipassEnv()
	if err != nil {
		return fmt.Errorf("failed to initialize multipass environment: %w", err)
	}

	// Get all clusters
	vms, err := mp.GetK3sVMs()
	if err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	if len(vms) == 0 {
		return fmt.Errorf("no clusters found")
	}

	// Get kubeconfig for each cluster
	var kubeconfigs []string
	for _, vm := range vms {
		kubeconfig, err := k3s.GetKubeconfig(mp, vm.Name)
		if err != nil {
			fmt.Printf("Warning: Failed to get kubeconfig for %s: %v\n", vm.Name, err)
			continue
		}
		kubeconfigs = append(kubeconfigs, kubeconfig)
	}

	if len(kubeconfigs) == 0 {
		return fmt.Errorf("failed to get any kubeconfigs")
	}

	// Merge kubeconfigs
	mergedConfig, err := k3s.MergeKubeconfigs(kubeconfigs)
	if err != nil {
		return fmt.Errorf("failed to merge kubeconfigs: %w", err)
	}

	// Save or print the merged kubeconfig
	if outputFile != "" {
		// Ensure directory exists
		dir := filepath.Dir(outputFile)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		}

		// Write kubeconfig to file
		if err := os.WriteFile(outputFile, []byte(mergedConfig), 0644); err != nil {
			return fmt.Errorf("failed to write kubeconfig: %w", err)
		}

		fmt.Printf("Merged kubeconfig saved to: %s\n", outputFile)
	} else {
		// Print to stdout
		fmt.Println(mergedConfig)
	}

	return nil
}
