package patch

import (
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/docker/scripts"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
	File  string `long:"file" description:"File path"`
	Name  string `long:"name" short:"n" description:"Parameter name"`
	Title string `long:"title" short:"t" description:"Title"`
	Force bool   `long:"force" short:"f" description:"Force"`
}

func Create() {
	args := getArgs()

	filePath := args.File
	patchName := args.Name
	title := args.Title
	force := args.Force

	if filePath == "" {
		log.Fatal("The --file option is incorrect or not specified.")
	}

	scripts.CreatePatch(filePath, patchName, title, force)
}

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
	if len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		var err error
		_, err = flags.ParseArgs(args, argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	return args
}
