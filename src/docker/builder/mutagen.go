package builder

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/faradey/madock/src/paths"
)

func syncMutagen(projectName, containerName, usr string) {
	clearMutagen(projectName, containerName)
	cmd := exec.Command("mutagen", "sync", "create", "--name",
		strings.ToLower(projectName)+"-"+containerName+"-1",
		"--default-group-beta", usr,
		"--default-owner-beta", usr,
		"--sync-mode", "two-way-resolved",
		"--default-file-mode", "0664",
		"--default-directory-mode", "0755",
		"--symlink-mode", "posix-raw",
		"--ignore-vcs",
		"-i", "/pub/static",
		"-i", "/pub/media",
		"-i", "/generated",
		"-i", "/var/cache",
		"-i", "/var/view_preprocessed",
		"-i", "/var/page_cache",
		"-i", "/var/tmp",
		"-i", "/var/vendor",
		"-i", "/phpserver",
		"-i", "/.idea",
		paths.GetRunDirPath(),
		"docker://"+strings.ToLower(projectName)+"-"+containerName+"-1/var/www/html",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Synchronization enabled")
	}
}

func clearMutagen(projectName, containerName string) {
	cmd := exec.Command("mutagen", "sync", "terminate",
		projectName+"-"+containerName+"-1",
	)
	cmd.Run()
}
