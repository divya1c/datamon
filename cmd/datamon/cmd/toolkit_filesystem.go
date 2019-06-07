package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/karrick/godirwalk"

	"github.com/spf13/cobra"
)

var filesystemCmd = &cobra.Command{
	Use:     "filesystem",
	Aliases: []string{"fs"},
	Short:   "Scan a filesystem and run operations on files and directories",
	Long: "This command allows a set of operations such as " +
		"\n 1. move old files into a garbage collection folder or delete old files" +
		"\n 2. Delete Empty directories" +
		"\n 3. Generate a list of files and/or directories",
	Run: func(cmd *cobra.Command, args []string) {
		runValidation(cmd)
		setup()
		setupFunctions()
		fmt.Printf("len of fns:%d\n", len(perFileFunctions))
		var wg sync.WaitGroup
		wg.Add(1)
		go walk(fsOps.path, wg)
		wg.Wait()
	},
}

type filesystemOptions struct {
	gcDuration          time.Duration
	gcFolder            string
	deletionDuration    time.Duration
	emptyDirDuration    time.Duration
	listFiles           bool
	recordFileSize      bool // Size per file
	listDirectories     bool
	calculateDirSize    bool // Size per directory
	maxConcurrency      uint
	maxChildConcurrency uint
	outputFile          string
	path                string
}

var fsOps filesystemOptions

func addGCDuration(cmd *cobra.Command) string {
	flag := gcDuration
	cmd.Flags().DurationVar(&fsOps.gcDuration, flag, 0, "Set how many days back (greater than 0) the file "+
		"should have been last updated for it to be "+
		"garbage collected.")
	return flag
}
func addGCFolder(cmd *cobra.Command) string {
	flag := gcFolder
	field := &fsOps.gcFolder
	cmd.Flags().StringVar(field, flag, "", "Path to GC Folder.")
	return flag
}
func addDeletionDuration(cmd *cobra.Command) string {
	flag := deleteDuration
	cmd.Flags().DurationVar(&fsOps.deletionDuration, flag, 0, "Set how many days back (greater than 0) the"+
		" file/dir should have been last updated for it to be "+
		"deleted.")
	return flag
}
func addEmptyDirDuration(cmd *cobra.Command) string {
	flag := emptyDirDuration
	cmd.Flags().DurationVar(&fsOps.emptyDirDuration, flag, 0, "Set how many days back (greater than 0) the "+
		"dir should have been last updated for it to be "+
		"deleted.")
	return flag
}
func addListFiles(cmd *cobra.Command) string {
	flag := listFiles
	field := &fsOps.listFiles
	cmd.Flags().BoolVar(field, flag, false, "Generate a list of all the files in the destination folder.")
	return flag
}
func addListDir(cmd *cobra.Command) string {
	flag := listDirectories
	field := &fsOps.listDirectories
	cmd.Flags().BoolVar(field, flag, false, "Generate a list of directories in the destination folder.")
	return flag
}
func addRecordFileSize(cmd *cobra.Command) string {
	flag := recordFileSize
	field := &fsOps.recordFileSize
	cmd.Flags().BoolVar(field, flag, false, "Record the size of files along when listing them.")
	return flag
}
func addCalculateDirSize(cmd *cobra.Command) string {
	flag := calculateDirSize
	field := &fsOps.calculateDirSize
	cmd.Flags().BoolVar(field, flag, false, "Calculate the total size of a directory.")
	return flag
}
func addMaxConcurrency(cmd *cobra.Command) string {
	flag := maxConcurrency
	field := &fsOps.maxConcurrency
	cmd.Flags().UintVar(field, flag, 10, "Maximum number of concurrent dir walks. Ulimit might need "+
		"to be adjusted for higher concurrency.")
	return flag
}
func addMaxChildConcurrency(cmd *cobra.Command) string {
	flag := maxChildConcurrency
	field := &fsOps.maxChildConcurrency
	cmd.Flags().UintVar(field, flag, 10, "Maximum number of concurrent children to be processed. Ulimit might need "+
		"to be adjusted for higher concurrency.")
	return flag
}
func addOutputFile(cmd *cobra.Command) string {
	flag := output
	field := &fsOps.outputFile
	cmd.Flags().StringVar(field, flag, "", "Path to output file for listing of files and or directories.")
	return flag
}
func addPath(cmd *cobra.Command) string {
	flag := path
	field := &fsOps.path
	cmd.Flags().StringVar(field, flag, "", "Target folder to process.")
	return flag
}

var walkCC chan struct{}
var childCC chan struct{}

func init() {
	addGCDuration(filesystemCmd)
	addGCFolder(filesystemCmd)
	addDeletionDuration(filesystemCmd)
	addEmptyDirDuration(filesystemCmd)
	addListDir(filesystemCmd)
	addListFiles(filesystemCmd)
	addRecordFileSize(filesystemCmd)
	addCalculateDirSize(filesystemCmd)
	addMaxConcurrency(filesystemCmd)
	addOutputFile(filesystemCmd)
	addMaxChildConcurrency(filesystemCmd)

	requiredFlags := []string{addPath(filesystemCmd)}

	for _, flag := range requiredFlags {
		err := filesystemCmd.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	toolkitCmd.AddCommand(filesystemCmd)
}

func runValidation(cmd *cobra.Command) {
	validOp := false
	handleInvalidOp := func(message string) {
		_ = cmd.Usage()
		log.Fatalf(message)
	}
	if fsOps.gcDuration > 0 {
		if fsOps.gcFolder == "" {
			handleInvalidOp("Set the garbage collection folder\n")
		}
	}
	if (fsOps.deletionDuration > 0 && fsOps.gcDuration > 0) && (fsOps.deletionDuration < fsOps.gcDuration) {
		handleInvalidOp("Duration for deletion should be greater than duration for garbage collection\n")
	}
	if fsOps.deletionDuration > 0 || fsOps.gcDuration > 0 {
		validOp = true
	}
	if fsOps.listFiles || fsOps.listDirectories || fsOps.recordFileSize || fsOps.calculateDirSize {
		if fsOps.outputFile == "" {
			handleInvalidOp("output file needs to be set via --" + output)
		}
		validOp = true
	}
	if !validOp {
		handleInvalidOp("Failed to set a valid operation to perform")
	}
}

var endDirProcessingFunctions []func(string, os.FileInfo) (string, error)
var perFileFunctions []func(string, os.FileInfo) (string, error)
var perDirectoryFunctions []func(string, os.FileInfo) (string, error)

func setup() {
	walkCC = make(chan struct{}, fsOps.maxConcurrency)
	childCC = make(chan struct{}, fsOps.maxChildConcurrency)
}

func setupFunctions() {
	if fsOps.emptyDirDuration > 0 {
		endDirProcessingFunctions = append(endDirProcessingFunctions, processEmptyDir)
	}
	if fsOps.gcDuration > 0 {
		perFileFunctions = append(perFileFunctions, processFileGC)
	}
	if fsOps.deletionDuration > 0 {
		perFileFunctions = append(perFileFunctions, processFileDeletion)
	}
	if fsOps.listFiles {
		perFileFunctions = append(perFileFunctions, processFilePath)
	}
	if fsOps.listDirectories {
		perDirectoryFunctions = append(perDirectoryFunctions, processFilePath)
	}
	if fsOps.recordFileSize {
		perFileFunctions = append(perFileFunctions, processFileSize)
	}
}

func expired(timestamp time.Time, daysToGC time.Duration, daysToDelete time.Duration) (bool, bool) {
	// if file eligible for deletion, do not garbage collect
	if (daysToDelete > 0) && (deleteFile(timestamp, daysToDelete)) {
		return false, true
	}
	return (daysToGC > 0) && (time.Now().Sub(timestamp) > time.Duration(daysToGC*24*time.Hour)), false
}

func deleteFile(timestamp time.Time, days time.Duration) bool {
	return time.Now().Sub(timestamp) > time.Duration(days*24*time.Hour)
}

func processFilePath(path string, _ os.FileInfo) (string, error) {
	fmt.Printf("list: %s\n", path)
	return "List:" + path, nil
}

func processFileSize(path string, fileInfo os.FileInfo) (string, error) {
	return fmt.Sprintf("FileSize:%s:%d", path, fileInfo.Size()), nil
}

func processFileGC(path string, fileInfo os.FileInfo) (s string, err error) {
	gc, _ := expired(fileInfo.ModTime(), fsOps.gcDuration, fsOps.deletionDuration)
	if gc {
		destination := fsOps.gcFolder + "/" + path
		err = os.MkdirAll(filepath.Base(destination), 0666)
		if err != nil {
			fmt.Printf("failed to create path:%s for file: %s", filepath.Base(destination), path)
			return
		}
		err = os.Rename(path, destination)
		fmt.Printf("moved %s:%s to %s", path, fileInfo.ModTime().String(), destination)
	}
	return
}

func processFileDeletion(path string, fileInfo os.FileInfo) (s string, err error) {
	_, del := expired(fileInfo.ModTime(), fsOps.gcDuration, fsOps.deletionDuration)
	if del {
		err = os.Remove(path)
		if err != nil {
			fmt.Printf("failed to delete file:%s err:%s", path, err)
			return "", err
		}
		fmt.Printf("deleted file:%s", path)
	}
	return
}

func processEmptyDir(path string, _ os.FileInfo) (s string, err error) {
	err = os.Remove(path)
	if err != nil {
		fmt.Printf("failed to delete path: %s", path)
		return
	}
	fmt.Printf("Deleted empty dir: %s", path)
	return
}

type task struct {
	path       string
	numDirEnts int
	mutex      sync.Mutex
}

func (t *task) decrementAndGet() int {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.numDirEnts--
	i := t.numDirEnts
	return i
}

type child struct {
	parent *task
	dirEnt *godirwalk.Dirent
}

func postWalkProcessing(path string) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return
	}
	for _, fn := range endDirProcessingFunctions {
		_, _ = fn(path, fileInfo)
	}
	return
}

// Note: Concurrency control is to control load on filesystem vs. controlling number of go routines.
func walk(path string, wg sync.WaitGroup) {
	childCC <- struct{}{}
	defer func() {
		<-childCC
		wg.Done()
	}()
	buffer := make([]byte, 1*1024*1024)
	dirEnts, err := godirwalk.ReadDirents(path, buffer)
	if err != nil {
		return
	}
	if dirEnts.Len() == 0 {
		postWalkProcessing(path)
		return
	}
	task := task{
		path:       path,
		numDirEnts: dirEnts.Len(),
		mutex:      sync.Mutex{},
	}
	for _, dirEnt := range dirEnts {
		wg.Add(1)
		go processChild(&child{
			parent: &task,
			dirEnt: dirEnt,
		}, wg)
	}
	fmt.Printf("walk %s done\n", path)
}

func processChild(child *child, wg sync.WaitGroup) {
	walkCC <- struct{}{}
	defer func() {
		<-walkCC
		wg.Done()
	}()
	path := filepath.Clean(child.parent.path + "/" + child.dirEnt.Name())
	statInfo, err := os.Stat(path)
	if err != nil {
		fmt.Printf("failed to process path:%s err:%s\n", path, err)
		return
	}
	if statInfo.IsDir() {
		wg.Add(1)
		go walk(path, wg)
	} else if statInfo.Mode().IsRegular() {
		for _, fn := range perFileFunctions {
			_, _ = fn(path, statInfo)
		}
	} else {
		fmt.Printf("skipping dirent:%s", path)
	}
	i := child.parent.decrementAndGet()
	if i == 0 {
		for _, fn := range perDirectoryFunctions {
			_, _ = fn(child.dirEnt.Name(), statInfo)
		}
		postWalkProcessing(child.parent.path)
		return
	}
}
