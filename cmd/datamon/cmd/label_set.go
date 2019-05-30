package cmd

import (
	"context"
	"fmt"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/spf13/cobra"
)

var SetLabelCommand = &cobra.Command{
	Use:   "set",
	Short: "Set labels",
	Long:  "Set the label corresponding to a bundle",
	Run: func(cmd *cobra.Command, args []string) {
		if repoParams.ContributorEmail == "" {
			logFatalln(fmt.Errorf("contributor email must be set in config or as a cli param"))
		}
		if repoParams.ContributorName == "" {
			logFatalln(fmt.Errorf("contributor name must be set in config or as a cli param"))
		}

		metaStore, err := gcs.New(repoParams.MetadataBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}

		bundle := core.New(core.NewBDescriptor(),
			core.Repo(repoParams.RepoName),
			core.MetaStore(metaStore),
			core.BundleID(bundleOptions.ID),
		)
		bundleExists, err := bundle.Exists(context.Background())
		if err != nil {
			logFatalln(err)
		}
		if !bundleExists {
			logFatalln(fmt.Errorf("bundle %v not found", bundle))
		}

		contributors := []model.Contributor{{
			Name:  repoParams.ContributorName,
			Email: repoParams.ContributorEmail,
		}}

		labelDescriptor := core.NewLabelDescriptor(
			core.LabelContributors(contributors),
		)
		label := core.NewLabel(labelDescriptor,
			core.LabelName(labelOptions.Name),
		)
		err = label.UploadDescriptor(context.Background(), bundle)
		if err != nil {
			logFatalln(err)
		}

	},
}

func init() {
	requiredFlags := []string{addRepoNameOptionFlag(SetLabelCommand)}

	/*
		requiredFlags = append(requiredFlags, addLabelNameFlag(SetLabelCommand))
		requiredFlags = append(requiredFlags, addBundleFlag(SetLabelCommand))
	*/

	addLabelNameFlag(SetLabelCommand)
	addBundleFlag(SetLabelCommand)

	for _, flag := range requiredFlags {
		err := SetLabelCommand.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	labelCmd.AddCommand(SetLabelCommand)
}