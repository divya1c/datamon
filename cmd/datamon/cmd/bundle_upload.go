package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/dlogger"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	fileUploadsByConcurrencyFactor = 5
)

// uploadBundleCmd is the command to upload a bundle from Datamon and model it locally.
var uploadBundleCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a bundle",
	Long:  "Upload a bundle consisting of all files stored in a directory",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		contributor, err := paramsToContributor(params)
		if err != nil {
			logFatalln(err)
		}
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			logFatalln(err)
		}
		sourceStore, err := paramsToSrcStore(ctx, params, false)
		if err != nil {
			logFatalln(err)
		}
		logger, err := dlogger.GetLogger(params.root.logLevel)
		if err != nil {
			logFatalln("Failed to set log level:" + err.Error())
		}
		bd := core.NewBDescriptor(
			core.Message(params.bundle.Message),
			core.Contributor(contributor),
		)
		bundle := core.New(bd,
			core.Repo(params.repo.RepoName),
			core.BlobStore(remoteStores.blob),
			core.ConsumableStore(sourceStore),
			core.MetaStore(remoteStores.meta),
			core.SkipMissing(params.bundle.SkipOnError),
			core.ConcurrentFileUploads(params.bundle.ConcurrencyFactor/fileUploadsByConcurrencyFactor),
			core.Logger(logger),
		)

		if params.bundle.FileList != "" {
			getKeys := func() ([]string, error) {
				var file afero.File
				file, err = os.Open(params.bundle.FileList)
				if err != nil {
					return nil, fmt.Errorf("failed to open file: %s err:%s", params.bundle.FileList, err.Error())
				}
				lineScanner := bufio.NewScanner(file)
				files := make([]string, 0)
				for lineScanner.Scan() {
					files = append(files, lineScanner.Text())
				}
				return files, nil
			}
			err = core.UploadSpecificKeys(ctx, bundle, getKeys)
		} else {
			err = core.Upload(ctx, bundle)
		}
		if err != nil {
			logFatalln(err)
		}
		log.Printf("Uploaded bundle id:%s ", bundle.BundleID)

		if params.label.Name != "" {
			labelDescriptor := core.NewLabelDescriptor(
				core.LabelContributor(contributor),
			)
			label := core.NewLabel(labelDescriptor,
				core.LabelName(params.label.Name),
			)
			err = label.UploadDescriptor(ctx, bundle)
			if err != nil {
				logFatalln(err)
			}
			log.Printf("set label '%v'", params.label.Name)
		}
	},
}

func init() {

	requiredFlags := []string{addRepoNameOptionFlag(uploadBundleCmd)}
	requiredFlags = append(requiredFlags, addPathFlag(uploadBundleCmd))
	requiredFlags = append(requiredFlags, addCommitMessageFlag(uploadBundleCmd))
	addFileListFlag(uploadBundleCmd)
	addLabelNameFlag(uploadBundleCmd)
	addSkipMissingFlag(uploadBundleCmd)
	addConcurrencyFactorFlag(uploadBundleCmd)
	addLogLevel(uploadBundleCmd)
	for _, flag := range requiredFlags {
		err := uploadBundleCmd.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	bundleCmd.AddCommand(uploadBundleCmd)
}
