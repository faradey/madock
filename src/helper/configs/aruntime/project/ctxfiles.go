package project

import (
	"os"
	"path/filepath"

	"github.com/faradey/madock/v4/src/helper/logger"
)

// Extension point for generated files inside a project's `ctx/`.
//
// Everything in `ctx/` is written by this package from a template in `docker/`,
// which is right for what the platforms need and leaves no way for the paid
// edition to add a file of its own — a logging fragment for nginx, say, or
// anything else mounted into a container. Without a seam the only options are a
// template nobody in this repository renders, or writing the file from a hook
// and hoping it runs before compose does.
//
// A provider that answers `false` means "this file should not exist", and the
// file is removed. That half matters as much as writing: the mount that uses it
// is conditional on the same setting, and a file left behind from an earlier
// configuration points a container at something that is no longer there.
//
// Nothing in this repository registers a provider, so `ctx/` holds exactly what
// it held before the seam existed.
type CtxFile struct {
	// Name is the file's name inside ctx/, without a directory.
	Name string

	// Render returns the file's content, or false to say it must not exist.
	Render func(projectName string) (string, bool)
}

var ctxFiles []CtxFile

// RegisterCtxFile adds a generated file to every project's ctx directory.
// Extension point for madock-pro. Registration order is write order.
func RegisterCtxFile(f CtxFile) {
	if f.Name == "" || f.Render == nil {
		return
	}
	ctxFiles = append(ctxFiles, f)
}

// writeCtxFiles is called at the end of the generation, so a provider sees a
// finished ctx directory and can be sure nothing overwrites it afterwards.
func writeCtxFiles(projectName, ctxDir string) {
	for _, file := range ctxFiles {
		target := filepath.Join(ctxDir, file.Name)

		content, wanted := file.Render(projectName)
		if !wanted {
			if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
				logger.Fatalln("removing " + target + ": " + err.Error())
			}

			continue
		}

		if err := os.WriteFile(target, []byte(content), 0644); err != nil {
			logger.Fatalln("writing " + target + ": " + err.Error())
		}
	}
}
