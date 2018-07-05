// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get the status of a bundle",
	Long: `Get the status of a bundle.

This command gives an overview of the uncommitted changes in a bundle.
So you see each file that's enlisted with its status (added, updated, removed).
`,
	Aliases: []string{"st"},
	Run: func(cmd *cobra.Command, args []string) {
		_, repo, err := initNamedRepo()
		if err != nil {
			log.Fatalln(err)
		}

		entries, err := repo.Stage().Status()
		if err != nil {
			log.Fatalln(err)
		}

		for _, entry := range entries {
			// TODO: do something less braindead than printing the path
			// stuff like A/M/D or +/x/- come to mind to indicate changes
			fmt.Println(entry.Path)
		}
	},
}

func init() {
	bundleCmd.AddCommand(statusCmd)
	addRepoFlag(statusCmd)

}