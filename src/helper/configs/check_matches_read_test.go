package configs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// readProjectEnv makes this test binary, started again by the test below, read
// one project's config the way every command does and exit. GetProjectConfigOnly
// ends in Fatalln (or, before configFromMapping, a panic) on a file it cannot
// read, so whether the read survives can only be seen from outside the process.
const readProjectEnv = "MADOCK_TEST_READ_PROJECT"

func TestMain(m *testing.M) {
	if name := os.Getenv(readProjectEnv); name != "" {
		CleanCache()
		GetProjectConfigOnly(name)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// TestTheCheckFailsExactlyWhenTheReadWould holds ProjectConfigParses to the read
// it stands in front of. The shared proxy builder, `stop` and GetProjectName call
// the check and then read; a check that passes where the read dies is how one
// broken .madock/config.xml took down the start of every other project — the
// first version read an unscoped `path` and never looked at the project's own
// file at all. Each case reads for real in a child process and compares.
func TestTheCheckFailsExactlyWhenTheReadWould(t *testing.T) {
	const good = "<config><scopes><default><marker>yes</marker></default></scopes></config>"
	const broken = "<config><scopes><default>not closed"

	cases := []struct {
		name        string
		target      string // project whose config is read
		runtime     string // its runtime config; "" = no file. SRC is replaced with its source dir
		own         string // the source dir's .madock/config.xml; "" = no file
		cwdOwn      string // the current directory's .madock/config.xml; "" = no file
		wantFailure bool
	}{
		{name: "both files good", target: "target",
			runtime: "<config><scopes><default><path>SRC</path></default></scopes></config>", own: good},
		{name: "runtime file broken", target: "target",
			runtime: broken, own: good, wantFailure: true},
		{name: "own .madock/config.xml broken", target: "target",
			runtime: "<config><scopes><default><path>SRC</path></default></scopes></config>", own: broken, wantFailure: true},
		{name: "path only in the active scope, own file broken", target: "target",
			runtime: "<config><activeScope>dev</activeScope><scopes><dev><path>SRC</path></dev></scopes></config>", own: broken, wantFailure: true},
		{name: "own file with an empty root", target: "target",
			runtime: "<config><scopes><default><path>SRC</path></default></scopes></config>", own: "<config>\n  \n</config>"},
		{name: "own file with text under the root", target: "target",
			runtime: "<config><scopes><default><path>SRC</path></default></scopes></config>", own: "<config>junk</config>", wantFailure: true},
		{name: "current project, no runtime file, own file in the current directory broken", target: "cur",
			cwdOwn: broken, wantFailure: true},
		{name: "another project with no path is not read from the current directory", target: "target",
			runtime: "<config><scopes><default><marker>yes</marker></default></scopes></config>", cwdOwn: broken},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			execDir := filepath.Join(root, "exec")
			runDir := filepath.Join(root, "run", "cur")
			srcDir := filepath.Join(root, "src")
			write := func(path, body string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			for _, dir := range []string{execDir, runDir, srcDir} {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if tc.runtime != "" {
				body := tc.runtime
				for i := 0; i+3 <= len(body); i++ {
					if body[i:i+3] == "SRC" {
						body = body[:i] + srcDir + body[i+3:]
						break
					}
				}
				write(filepath.Join(execDir, "projects", tc.target, "config.xml"), body)
			}
			if tc.own != "" {
				write(filepath.Join(srcDir, ".madock", "config.xml"), tc.own)
			}
			if tc.cwdOwn != "" {
				write(filepath.Join(runDir, ".madock", "config.xml"), tc.cwdOwn)
			}

			t.Setenv("MADOCK_EXEC_DIR", execDir)
			t.Setenv("MADOCK_RUN_DIR", runDir)
			CleanCache()
			t.Cleanup(CleanCache)

			checkErr := ProjectConfigParses(tc.target)

			child := exec.Command(os.Args[0], "-test.run=^$")
			child.Env = append(os.Environ(), readProjectEnv+"="+tc.target)
			out, readErr := child.CombinedOutput()
			readFailed := readErr != nil

			if readFailed != tc.wantFailure {
				t.Fatalf("the read itself: failed=%v, the case expects %v — the case is wrong, not the check\n%s",
					readFailed, tc.wantFailure, out)
			}
			if (checkErr != nil) != readFailed {
				t.Errorf("check error = %v, but the read failed=%v\n%s", checkErr, readFailed, out)
			}
		})
	}
}

// configFromMapping on its own: an empty root is a config that sets nothing,
// text under the root is an error, and neither may panic.
func TestConfigFromMappingTellsAnEmptyRootFromText(t *testing.T) {
	for body, wantErr := range map[string]bool{
		"<config/>":                 false,
		"<config></config>":         false,
		"<config>\n \t\n</config>":  false,
		"<config>junk</config>":     true,
		"<config><a>1</a></config>": false,
		"<other><a>1</a></other>":   false,
	} {
		mapping, err := GetXmlMapFromBytes([]byte(body))
		if err != nil {
			t.Fatalf("%q did not decode: %v", body, err)
		}
		_, err = configFromMapping(mapping)
		if (err != nil) != wantErr {
			t.Errorf("%q: err = %v, want error %v", body, err, wantErr)
		}
	}
}
