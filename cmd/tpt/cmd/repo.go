// Copyright © 2018 One Concern

package cmd

import (
	"github.com/spf13/cobra"
)

var repoOptions struct {
	Name        string
	Description string
}

// repoCmd represents the repo command
var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Data Repo management related operations",
	Long: `Data repository management related operations for trumpet.

Repositories don't carry much content until a commit is made.
`,
}

func init() {
	rootCmd.AddCommand(repoCmd)

}

func addRepoOptions(cmd *cobra.Command) error {
	fls := cmd.Flags()
	if err := addRepoNameOption(cmd); err != nil {
		return err
	}
	fls.StringVar(&repoOptions.Description, "description", "", "A description of this repository")
	return nil
}

func addRepoNameOption(cmd *cobra.Command) error {
	fls := cmd.Flags()
	fls.StringVar(&repoOptions.Name, "name", "", "The name of this repository")
	return cmd.MarkFlagRequired("name")
}
