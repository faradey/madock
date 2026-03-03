package embedded

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/faradey/madock/v3/src/helper/paths"
)

var DockerFS fs.FS
var ScriptsFS fs.FS

func SetDockerFS(f fs.FS) {
	DockerFS = f
}

func SetScriptsFS(f fs.FS) {
	ScriptsFS = f
}

// ExtractIfNeeded extracts embedded assets to disk when version changes.
func ExtractIfNeeded(appVersion string) {
	execDir := paths.GetExecDirPath()
	markerFile := filepath.Join(execDir, ".embedded_version")

	existing, _ := os.ReadFile(markerFile)
	if string(existing) == appVersion {
		return
	}

	if DockerFS != nil {
		extractFS(DockerFS, filepath.Join(execDir, "docker"))
	}
	if ScriptsFS != nil {
		extractFS(ScriptsFS, filepath.Join(execDir, "scripts"))
	}

	os.WriteFile(markerFile, []byte(appVersion), 0644)
}

func extractFS(fsys fs.FS, destDir string) {
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == "." {
			return err
		}
		target := filepath.Join(destDir, path)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		os.MkdirAll(filepath.Dir(target), 0755)
		return os.WriteFile(target, data, 0755)
	})
}
