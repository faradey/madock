package paths

import (
	"os"
	"path/filepath"
	"testing"
)

// The middle case is the one that differed between platforms, and the reason
// this function moved: a directory holding only empty directories was "empty"
// on Medusa and "occupied" on the other five, so `setup` refused on five
// platforms and proceeded on the sixth, over the same directory.
func TestIsDirEmpty(t *testing.T) {
	cases := []struct {
		name  string
		build func(t *testing.T, dir string)
		want  bool
		why   string
	}{
		{
			name:  "nothing in it",
			build: func(*testing.T, string) {},
			want:  true,
			why:   "the ordinary case",
		},
		{
			name: "only madock's own directory",
			build: func(t *testing.T, dir string) {
				mkdir(t, filepath.Join(dir, ".madock"))
				write(t, filepath.Join(dir, ".madock", "config.xml"), "<config/>")
			},
			want: true,
			why:  "setup started and did not finish; refusing here strands the person",
		},
		{
			name: "empty directories, several deep",
			build: func(t *testing.T, dir string) {
				mkdir(t, filepath.Join(dir, "src", "app", "components"))
				mkdir(t, filepath.Join(dir, "public"))
			},
			want: true,
			why:  "an empty tree holds nothing to lose — true on Medusa and false everywhere else",
		},
		{
			name: "one file, however deep",
			build: func(t *testing.T, dir string) {
				mkdir(t, filepath.Join(dir, "src", "app"))
				write(t, filepath.Join(dir, "src", "app", "page.tsx"), "export default null")
			},
			want: false,
			why:  "this is the whole point: somebody's work must not be installed over",
		},
		{
			name: "a file at the top",
			build: func(t *testing.T, dir string) {
				write(t, filepath.Join(dir, "README.md"), "mine")
			},
			want: false,
			why:  "the obvious case, and the only one all six copies agreed on",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			c.build(t, dir)

			if got := IsDirEmpty(dir); got != c.want {
				t.Errorf("IsDirEmpty = %v, want %v — %s", got, c.want, c.why)
			}
		})
	}
}

// A directory that cannot be read answers "empty".
//
// The permissive direction, and what every copy already did: refusing would stop
// setup on a permissions problem, with a message about the directory not being
// empty that names nothing the person can act on.
func TestIsDirEmptyAnswersEmptyWhenItCannotLook(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads unreadable directories, so this cannot be staged")
	}

	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	mkdir(t, locked)
	write(t, filepath.Join(locked, "file"), "content")

	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	if !IsDirEmpty(locked) {
		t.Error("an unreadable directory was reported as occupied")
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
