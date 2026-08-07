package restore

import (
	"bufio"
	"compress/gzip"
	"fmt"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/controller/general/rebuild"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ArgsStruct mirrors project:remove, which is the other command here that
// destroys something and therefore has to be usable both ways.
type ArgsStruct struct {
	attr.Arguments
	Name string `arg:"-n,--name" help:"Snapshot name to restore. Without it, choose from a list"`
}

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"snapshot:restore"},
		Handler:  Execute,
		Help:     "Restore snapshot",
		Category: "snapshot",
		ArgsType: new(ArgsStruct),
	})
}

func Execute() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)

	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	dbsPath := paths.GetExecDirPath() + "/projects/" + projectName + "/backup/snapshot"
	var snapshotNames []string
	if paths.IsFileExist(dbsPath) {
		snapshotNames = paths.GetDirs(dbsPath)
	}

	if len(snapshotNames) == 0 {
		logger.Fatal("No snapshots found for restore")
	}

	selectedInt := 0

	// A named snapshot skips the list entirely. Restoring is exactly the sort of
	// thing that ends up in a script — a nightly reset of a demo, a step in a
	// runbook — and until this existed the only way in was to type a number at a
	// prompt, which no script can do.
	if args.Name != "" {
		// Two things can be meant by a name, because `snapshot:create -n backup`
		// stores it as `snapshot-backup-2026-08-08-00-58-12`. An exact directory
		// name is unambiguous and wins. Otherwise the name is read as the one
		// given at creation, and the newest snapshot carrying it is restored —
		// the timestamp suffix sorts chronologically, so "newest" is the last.
		wanted := "snapshot-" + args.Name + "-"
		matched := ""
		for _, snapshotName := range snapshotNames {
			base := filepath.Base(snapshotName)
			if base == args.Name {
				matched = base
				break
			}
			if strings.HasPrefix(base, wanted) && base > matched {
				matched = base
			}
		}

		for index, snapshotName := range snapshotNames {
			if filepath.Base(snapshotName) == matched {
				selectedInt = index + 1
				break
			}
		}

		// Said out loud when the name was a prefix: the user asked for "backup"
		// and something dated is about to replace their database.
		if selectedInt > 0 && matched != args.Name {
			fmt.Println("Restoring \"" + matched + "\"")
		}

		if selectedInt == 0 {
			fmt.Println("Snapshots that do exist:")
			for _, snapshotName := range snapshotNames {
				fmt.Println("  " + filepath.Base(snapshotName))
			}
			// Named and not found is a mistake worth stopping on. Falling back
			// to the prompt would hang a script; picking a different snapshot
			// would restore the wrong data.
			logger.Fatal("No snapshot named \"" + args.Name + "\"")
		}
	} else {
		for index, snapshotName := range snapshotNames {
			fmt.Println(strconv.Itoa(index+1) + ") " + filepath.Base(snapshotName))
		}

		fmt.Println("Choose one of the offered variants")
		buf := bufio.NewReader(os.Stdin)
		sentence, err := buf.ReadBytes('\n')
		selected := strings.TrimSpace(string(sentence))
		if err != nil {
			logger.Fatalln(err)
		}
		selectedInt, err = strconv.Atoi(selected)
		if err != nil || selectedInt < 1 || selectedInt > len(snapshotNames) {
			logger.Fatal("The item you selected was not found")
		}
	}

	RestoreSnapshot(projectName, projectConf, selectedInt, snapshotNames, dbsPath)
	os.Args = append(os.Args, "-c")
	rebuild.Execute()
	fmt.Println("Snapshot restored successfully")
}

func RestoreSnapshot(projectName string, projectConf map[string]string, selectedInt int, snapshotNames []string, dbsPath string) {
	containerName := docker.GetContainerName(projectConf, projectName, "snapshot")
	docker.Down(projectName, false)
	docker.UpSnapshot(projectName)
	if paths.IsFileExist(dbsPath + "/" + snapshotNames[selectedInt-1] + "/db.tar.gz") {
		selectedFile, err := os.Open(dbsPath + "/" + snapshotNames[selectedInt-1] + "/db.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
		defer selectedFile.Close()
		cmd, prepErr := docker.PrepareContainerExec(containerName, "root", false, "bash", "-c", "rm -rf /var/www/mysql/* && cd /var/www/mysql && tar -zxf -")
		if prepErr != nil {
			logger.Fatal(prepErr)
		}
		out, err := gzip.NewReader(selectedFile)
		if err != nil {
			logger.Fatal(err)
		}
		defer out.Close()
		cmd.Stdin = out
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err, containerName)
		}
	}

	if projectConf["db2/enabled"] == "true" && paths.IsFileExist(dbsPath+"/"+snapshotNames[selectedInt-1]+"/db2.tar.gz") {
		selectedFileDb2, err := os.Open(dbsPath + "/" + snapshotNames[selectedInt-1] + "/db2.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
		defer selectedFileDb2.Close()
		cmd, prepErr := docker.PrepareContainerExec(containerName, "root", false, "bash", "-c", "rm -rf /var/www/mysql2/mysql/* && cd /var/www/mysql2/mysql && tar -zxf -")
		if prepErr != nil {
			logger.Fatal(prepErr)
		}
		outDb2, err := gzip.NewReader(selectedFileDb2)
		if err != nil {
			logger.Fatal(err)
		}
		defer outDb2.Close()
		cmd.Stdin = outDb2
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err, containerName)
		}
	}

	if paths.IsFileExist(dbsPath + "/" + snapshotNames[selectedInt-1] + "/files.tar.gz") {
		selectedFileFiles, err := os.Open(dbsPath + "/" + snapshotNames[selectedInt-1] + "/files.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
		defer selectedFileFiles.Close()
		cmd, prepErr := docker.PrepareContainerExec(containerName, "root", false, "bash", "-c", "rm -rf /var/www/html/* && cd /var/www/html && tar -zxf -")
		if prepErr != nil {
			logger.Fatal(prepErr)
		}
		outFiles, err := gzip.NewReader(selectedFileFiles)
		if err != nil {
			logger.Fatal(err)
		}
		defer outFiles.Close()
		cmd.Stdin = outFiles
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err, containerName)
		}
	}

	docker.StopSnapshot(projectName)
}
