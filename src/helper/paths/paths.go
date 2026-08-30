package paths

import (
	"fmt"
	"github.com/faradey/madock/v4/src/helper/hash"
	"github.com/faradey/madock/v4/src/helper/logger"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// PathValidator allows enterprise to validate env var overrides for critical paths.
type PathValidator func(envName, value string) error

var pathValidator PathValidator

// SetPathValidator installs the validator for MADOCK_* environment overrides.
//
// Extension point for madock-pro: its pathguard hook registers here, and that is
// what stops a destructive command being pointed at a directory it should not
// reach. Unreachable from this module by design.
func SetPathValidator(v PathValidator) {
	pathValidator = v
}

func getEnvWithValidation(envName string) string {
	val := os.Getenv(envName)
	if val != "" && pathValidator != nil {
		if err := pathValidator(envName, val); err != nil {
			logger.Fatalln("Invalid env override " + envName + ": " + err.Error())
		}
	}
	return val
}

func GetExecDirPath() string {
	if envDir := getEnvWithValidation("MADOCK_EXEC_DIR"); envDir != "" {
		return envDir
	}

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
	if envDir := getEnvWithValidation("MADOCK_RUN_DIR"); envDir != "" {
		return envDir
	}

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

// GetDirs lists the directories inside a path, following symlinks.
//
// os.ReadDir answers about the entry, not its target, so a symlink to a directory
// has IsDir() false and used to be dropped here. Registry entries are symlinks on
// every machine that sets a project up from a temporary checkout — a cluster VM had
// four of them — and everything that walks projects/ goes through this function:
// the migrations, project:clone, and the project list. On such a machine
// `project:list` answered "No projects are registered" while four were running,
// which reads as "all clean" rather than as "I cannot see".
//
// A broken symlink fails the Stat and is skipped, which is what should happen to a
// registry entry pointing at nothing.
func GetDirs(path string) (dirs []string) {
	items, err := os.ReadDir(path)
	if err != nil {
		logger.Fatal(err)
	}

	for _, file := range items {
		info, statErr := os.Stat(filepath.Join(path, file.Name()))
		if statErr != nil || !info.IsDir() {
			continue
		}
		dirs = append(dirs, file.Name())
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
	active, err := ActiveProjects()
	if err != nil {
		logger.Println(err)
	}

	return active
}

// ActiveProjects reports which registered projects have containers running, and
// says so when it could not find out.
//
// GetActiveProjects above swallows that difference, which is right for its
// callers — `stop` uses it to decide whether the shared proxy is still needed,
// and there "cannot ask docker" and "nothing is running" lead to the same place.
// It is wrong for anything that reports to a person: "no projects are running"
// and "docker did not answer" are different facts, and printing the first when
// the second is true is the kind of confident wrong answer this codebase keeps
// finding.
//
// One `docker ps` for the whole registry rather than a status call per project.
func ActiveProjects() ([]string, error) {
	cmd := exec.Command("docker", "ps", "--format", "json")
	// Output, not CombinedOutput: docker's warnings go to stderr, and a project
	// whose name appeared in one would match below.
	var stderr strings.Builder
	cmd.Stderr = &stderr
	result, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps failed: %w\n%s", err, stderr.String())
	}

	var activeProjects []string
	resultString := string(result)
	projects := GetDirs(MakeDirsByPath(RuntimeProjects()))
	for _, projectName := range projects {
		if strings.Contains(resultString, strings.ToLower(projectName)+"-") {
			activeProjects = append(activeProjects, projectName)
		}
	}

	return activeProjects, nil
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
