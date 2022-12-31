package commands

import (
	"log"

	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/docker/scripts"
)

func PatchCreate() {
	filePath := attr.Options.File
	patchName := attr.Options.Name
	if filePath == "" {
		log.Fatal("The --file option is incorrect or not specified.")
	}

	if patchName == "" {
		log.Fatal("The --name option is incorrect or not specified.")
	}

	scripts.CreatePatch(filePath, patchName)
}
