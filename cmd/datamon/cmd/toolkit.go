package cmd

import "github.com/spf13/cobra"

var toolkitCmd = &cobra.Command{
	Use:     "toolkit",
	Aliases: []string{"tk"},
	Short:   "Collection of operations to run related to Datamon.",
	Long:    "Collection of miscellaneous operations that are needed around Datamon workflows.",
}

func init() {
	rootCmd.AddCommand(toolkitCmd)
}
