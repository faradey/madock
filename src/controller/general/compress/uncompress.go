package compress

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/paths"
)

func Unzip() {
	attr.Parse(new(ArgsStruct))

	isOk := false
	basePath := paths.GetRunDirPath() + "/"
	r, err := zip.OpenReader(basePath + archiveName)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Fatal(err)
		}
		if isOk {
			os.Remove(basePath + archiveName)
		}
	}()

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		path := filepath.Join(basePath, f.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(basePath)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			err := os.MkdirAll(path, 0775)
			if err != nil {
				return err
			}
		} else {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer func() {
				if err := rc.Close(); err != nil {
					log.Fatal(err)
				}
			}()
			/*fmt.Println(filepath.Dir(path))
			fmt.Println(f.Mode())
			err := os.MkdirAll(filepath.Dir(path), f.Mode())
			if err != nil {
				return err
			}*/
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					log.Fatal(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	isOk = true
}
