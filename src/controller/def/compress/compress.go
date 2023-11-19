package compress

import (
	"archive/zip"
	"fmt"
	"github.com/faradey/madock/src/paths"
	"os"
)

var archiveName string = "madock_compressed_project.zip"

func Zip() {
	baseFolder := paths.GetRunDirPath() + "/"
	//fmt.Println(baseFolder + "/" + archiveName)
	//return
	// Get a Buffer to Write To
	outFile, err := os.Create(baseFolder + archiveName)
	if err != nil {
		fmt.Println(err)
	}
	defer outFile.Close()

	// Create a new zip archive.
	w := zip.NewWriter(outFile)

	// Add some files to the archive.
	addFiles(w, baseFolder, "")

	if err != nil {
		fmt.Println(err)
	}

	// Make sure to check the error on Close.
	err = w.Close()
	if err != nil {
		fmt.Println(err)
	}
	removeFiles(baseFolder)
}

func addFiles(w *zip.Writer, basePath, baseInZip string) {
	// Open the Directory
	files, err := os.ReadDir(basePath)
	if err != nil {
		fmt.Println(err)
	}

	for _, file := range files {
		fmt.Println(basePath + file.Name())
		if !file.IsDir() {
			if file.Name() != archiveName {
				dat, err := os.ReadFile(basePath + file.Name())
				if err != nil {
					fmt.Println(err)
				}

				// Add some files to the archive.
				f, err := w.Create(baseInZip + file.Name())
				if err != nil {
					fmt.Println(err)
				}
				_, err = f.Write(dat)
				if err != nil {
					fmt.Println(err)
				}
			}
		} else if file.IsDir() {
			// Recurse
			newBase := basePath + file.Name() + "/"
			fmt.Println("Recursing and Adding SubDir: " + file.Name())
			fmt.Println("Recursing and Adding SubDir: " + newBase)

			// Add some files to the archive.
			_, err := w.Create(baseInZip + file.Name() + "/")
			if err != nil {
				fmt.Println(err)
			}
			addFiles(w, newBase, baseInZip+file.Name()+"/")
		}
	}
}

func removeFiles(basePath string) {
	// Open the Directory
	files, err := os.ReadDir(basePath)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, file := range files {
		if !file.IsDir() {
			if file.Name() != archiveName {
				os.Remove(basePath + file.Name())
			}
		} else if file.IsDir() {
			newBase := basePath + file.Name() + "/"
			os.RemoveAll(newBase)
		}
	}
}
