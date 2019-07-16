// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	daemonizer "github.com/jacobsa/daemonize"

	"github.com/oneconcern/datamon/internal"

	"github.com/oneconcern/datamon/pkg/dlogger"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"

	"github.com/spf13/cobra"
)

func undaemonizeArgs(args []string) []string {
	foregroundArgs := make([]string, 0)
	for _, arg := range args {
		if arg != "--"+addDaemonizeFlag(nil) {
			foregroundArgs = append(foregroundArgs, arg)
		}
	}
	return foregroundArgs
}

func runDaemonized() {
	var path string
	path, err := os.Executable()
	if err != nil {
		err = fmt.Errorf("os.Executable: %v", err)
		logFatalln(err)
	}

	foregroundArgs := undaemonizeArgs(os.Args[1:])

	// Pass along PATH so that the daemon can find fusermount on Linux.
	env := []string{
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
	}

	// Pass along GOOGLE_APPLICATION_CREDENTIALS
	if p, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS"); ok {
		env = append(env, fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", p))
	}

	// Run.
	err = daemonizer.Run(path, foregroundArgs, env, os.Stdout)
	if err != nil {
		err = fmt.Errorf("daemonize.Run: %v", err)
		logFatalln(err)
	}
}

func onDaemonError(err error) {
	if errSig := daemonizer.SignalOutcome(err); errSig != nil {
		logFatalln(fmt.Errorf("error SignalOutcome: %v, cause: %v", errSig, err))
	}
	logFatalln(err)
}

// Mount a read only view of a bundle
var mountBundleCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount a bundle",
	Long:  "Mount a readonly, non-interactive view of the entire data that is part of a bundle",
	Run: func(cmd *cobra.Command, args []string) {
		if params.bundle.Daemonize {
			runDaemonized()
			return
		}

		var err error

		var consumableStorePath string
		if params.bundle.DataPath == "" {
			consumableStorePath, err = ioutil.TempDir("", "datamon-mount-destination")
			if err != nil {
				log.Fatalf("Couldn't create temporary directory: %v\n", err)
				return
			}
		} else {
			consumableStorePath, err = sanitizePath(params.bundle.DataPath)
			if err != nil {
				log.Fatalf("Failed to sanitize destination: %s\n", params.bundle.DataPath)
				return
			}
			createPath(consumableStorePath)
		}

		metadataSource, err := gcs.New(params.repo.MetadataBucket, config.Credential)
		if err != nil {
			onDaemonError(err)
		}
		blobStore, err := gcs.New(params.repo.BlobBucket, config.Credential)
		if err != nil {
			onDaemonError(err)
		}
		consumableStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), consumableStorePath))

		err = setLatestOrLabelledBundle(metadataSource)
		if err != nil {
			logFatalln(err)
		}
		bd := core.NewBDescriptor()
		bundle := core.New(bd,
			core.Repo(params.repo.RepoName),
			core.BundleID(params.bundle.ID),
			core.BlobStore(blobStore),
			core.ConsumableStore(consumableStore),
			core.MetaStore(metadataSource),
			core.Streaming(params.bundle.Stream),
		)
		logger, err := dlogger.GetLogger(params.root.logLevel)
		if err != nil {
			log.Fatalln("Failed to set log level:" + err.Error())
		}
		fs, err := core.NewReadOnlyFS(bundle, logger)
		if err != nil {
			onDaemonError(err)
		}
		if err = fs.MountReadOnly(params.bundle.MountPath); err != nil {
			onDaemonError(err)
		}

		registerSIGINTHandlerMount(params.bundle.MountPath)
		if err = daemonizer.SignalOutcome(nil); err != nil {
			logFatalln(err)
		}

//		mempollgoroutine(logger)

		func() {
			minMBs := make([]internal.MinProfMB, 0)
			for i := 12; i < 26; i++ {
				minMBs = append(minMBs, internal.MinProfMB{Alloc: uint64(i*1000)})
			}
			internal.MemPoll(internal.MemPollParams{
				Logger: logger,
				MinMBs: minMBs,
			})
		}()

		if err = fs.JoinMount(context.Background()); err != nil {
			logFatalln(err)
		}
	},
}

func init() {

	requiredFlags := []string{addRepoNameOptionFlag(mountBundleCmd)}
	addBucketNameFlag(mountBundleCmd)
	addDaemonizeFlag(mountBundleCmd)
	addBlobBucket(mountBundleCmd)
	addBundleFlag(mountBundleCmd)
	addLogLevel(mountBundleCmd)
	addStreamFlag(mountBundleCmd)
	addLabelNameFlag(mountBundleCmd)
	// todo: #165 add --cpuprof to all commands via root
	addCPUProfFlag(mountBundleCmd)
	addDataPathFlag(mountBundleCmd)
	requiredFlags = append(requiredFlags, addMountPathFlag(mountBundleCmd))

	for _, flag := range requiredFlags {
		err := mountBundleCmd.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	bundleCmd.AddCommand(mountBundleCmd)
}
