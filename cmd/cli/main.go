package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "featury",
	Short: "Featury CLI - Feature flag management tool",
	Long:  "A command-line interface for managing feature flags with the Featury service",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("featury CLI v1.0.0")
	},
}

var featureCmd = &cobra.Command{
	Use:   "feature",
	Short: "Manage features",
	Long:  "Create, update, delete and list feature flags",
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all features",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Listing all features...")
	},
}

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new feature",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		fmt.Printf("Creating feature: %s\n", name)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(featureCmd)
	featureCmd.AddCommand(listCmd)
	featureCmd.AddCommand(createCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}