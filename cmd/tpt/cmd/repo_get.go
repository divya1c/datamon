// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
	"log"

	"github.com/oneconcern/trumpet"
	"github.com/oneconcern/trumpet/pkg/store"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

type DataRepo struct {
	store.Repo    `json:",inline" yaml:",inline"`
	CurrentBranch string
}

// repoGetCmd represents the get command
var repoGetCmd = &cobra.Command{
	Use:   "get",
	Short: "get the details for a repository",
	Long:  `get the details for a repository as json`,
	Run: func(cmd *cobra.Command, args []string) {
		tpt, err := trumpet.New("")
		if err != nil {
			log.Fatalln(err)
		}

		repo, err := tpt.GetRepo(repoOptions.Name)
		if err != nil {
			log.Fatalln(err)
		}
		b, err := yaml.Marshal(repo)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(string(b))
	},
}

func init() {
	repoCmd.AddCommand(repoGetCmd)
	addRepoNameOption(repoGetCmd)
}
