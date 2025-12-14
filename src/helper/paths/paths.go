package paths

import (
	"github.com/faradey/madock/src/helper/hash"
	"github.com/faradey/madock/src/helper/logger"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func GetExecDirPath() string {
	var dirAbsPath string

	ex, err := os.Executable()
	if err != nil {
		logger.Fatal(err)
	}
	exReal, err := filepath.EvalSymlinks(ex)
	if err != nil {
		dirAbsPath = filepath.Dir(ex)
	} else {
		dirAbsPath = filepath.Dir(exReal)
	}

	return dirAbsPath
}

func GetExecDirName() string {
	return filepath.Base(GetExecDirPath())
}

func GetExecDirNameByPath(path string) string {
	return filepath.Base(path)
}

func GetRunDirPath() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	return dir
}

func GetRunDirName() string {
	return filepath.Base(GetRunDirPath())
}

func GetRunDirNameWithHash() string {
	return filepath.Base(GetRunDirPath()) + "__" + strconv.Itoa(int(hash.Hash(GetRunDirPath())))
}

func GetDirs(path string) (dirs []string) {
	items, err := os.ReadDir(path)
	if err != nil {
		logger.Fatal(err)
	}

	for _, file := range items {
		if file.IsDir() {
			dirs = append(dirs, file.Name())
		}
	}

	return dirs
}

func GetFiles(path string) (dirs []string) {
	items, err := os.ReadDir(path)
	if err != nil {
		logger.Fatal(err)
	}

	for _, file := range items {
		if !file.IsDir() {
			dirs = append(dirs, file.Name())
		}
	}

	return dirs
}

func GetFilesRecursively(path string) (dirs []string) {
	items, err := os.ReadDir(path)
	if err == nil {
		for _, file := range items {
			if !file.IsDir() {
				dirs = append(dirs, path+"/"+file.Name())
			} else {
				dirs = append(dirs, GetFilesRecursively(path+"/"+file.Name())...)
			}
		}
	}

	return dirs
}

func GetDBFiles(path string) (dirs []string) {
	items, err := os.ReadDir(path)
	if err != nil {
		logger.Fatal(err)
	}

	for _, file := range items {
		fileName := file.Name()
		if !file.IsDir() {
			if len(fileName) > 0 && !strings.HasPrefix(fileName, ".") &&
				strings.Contains(strings.ToLower(fileName), ".sql") &&
				!strings.Contains(strings.ToLower(path), "/dev/tests/acceptance") &&
				!strings.Contains(strings.ToLower(path), strings.ToLower(strings.Trim(GetRunDirPath(), "/"))+"/vendor/") {
				dirs = append(dirs, path+"/"+fileName)
			}
		} else {
			dirs = append(dirs, GetDBFiles(path+"/"+fileName)...)
		}
	}

	return dirs
}

func MakeDirsByPath(val string) string {
	trimVal := strings.Trim(val, "/")
	if trimVal != "" {
		dirs := strings.Split(trimVal, "/")
		var err error
		for i := 0; i < len(dirs); i++ {
			if !IsFileExist("/" + strings.Join(dirs[:i+1], "/")) {
				err = os.Mkdir("/"+strings.Join(dirs[:i+1], "/"), 0755)
				if err != nil {
					logger.Fatal(err)
				}
			}
		}
	}

	return val
}

func GetActiveProjects() []string {
	var activeProjects []string
	cmd := exec.Command("docker", "ps", "--format", "json")
	result, err := cmd.CombinedOutput()
	if err != nil {
		logger.Println(err, string(result))
	} else {
		resultString := string(result)
		projects := GetDirs(MakeDirsByPath(RuntimeProjects()))
		for _, projectName := range projects {
			if strings.Contains(resultString, strings.ToLower(projectName)+"-") {
				activeProjects = append(activeProjects, projectName)
			}
		}
	}

	return activeProjects
}

func IsFileExist(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	}

	return false
}

func Copy(srcpath, dstpath string) (err error) {
	r, err := os.Open(srcpath)
	if err != nil {
		return err
	}
	defer r.Close() // ignore error: file was opened read-only.

	w, err := os.Create(dstpath)
	if err != nil {
		return err
	}

	defer func() {
		// Report the error, if any, from Close, but do so
		// only if there isn't already an outgoing error.
		if c := w.Close(); err == nil {
			err = c
		}
	}()

	_, err = io.Copy(w, r)
	return err
}

func CopyDir(dst, src string) error {
	src, err := filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == src {
			return nil
		}

		if strings.HasPrefix(filepath.Base(path), ".") {
			// Skip any dot files
			if info.IsDir() {
				return filepath.SkipDir
			} else {
				return nil
			}
		}

		// The "path" has the src prefixed to it. We need to join our
		// destination with the path without the src on it.
		dstPath := filepath.Join(dst, path[len(src):])

		// we don't want to try and copy the same file over itself.
		if eq, err := SameFile(path, dstPath); eq {
			return nil
		} else if err != nil {
			return err
		}

		// If we have a directory, make that subdirectory, then continue
		// the walk.
		if info.IsDir() {
			if path == filepath.Join(src, dst) {
				// dst is in src; don't walk it.
				return nil
			}

			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return err
			}

			return nil
		}

		// If the current path is a symlink, recreate the symlink relative to
		// the dst directory
		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}

			return os.Symlink(target, dstPath)
		}

		// If we have a file, copy the contents.
		srcF, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcF.Close()

		dstF, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstF.Close()

		if _, err := io.Copy(dstF, srcF); err != nil {
			return err
		}

		// Chmod it
		return os.Chmod(dstPath, info.Mode())
	}

	return filepath.Walk(src, walkFn)
}

func SameFile(a, b string) (bool, error) {
	if a == b {
		return true, nil
	}

	aInfo, err := os.Lstat(a)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	bInfo, err := os.Lstat(b)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return os.SameFile(aInfo, bInfo), nil
}
