//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestProjectsStartedAtOnceKeepEveryHost starts several projects at once, as
// whoever drives madock from a queue does. proxy.conf is one file written whole
// from what its writer read, and without one writer at a time the last of them
// takes the others' hosts out. It is a race, so several rounds.
func TestProjectsStartedAtOnceKeepEveryHost(t *testing.T) {
	const rounds, projects = 6, 4

	install := newInstallation(t)
	all := make([]*project, projects)
	for i := range all {
		name := fmt.Sprintf("e2eat%d", i)
		all[i] = install.project(name)
		all[i].run(setupTimeout, "setup", "-y", "--platform=custom", "--language=none", "--hosts="+name+".test")
	}
	all[0].run(20*time.Minute, "start")

	for round := range rounds {
		var wg sync.WaitGroup
		hosts := make([]string, projects)
		for i, p := range all {
			hosts[i] = fmt.Sprintf("%s-r%d.example.com", p.name, round)
			wg.Add(1)
			go func() {
				defer wg.Done()
				if out, err := p.tryRun(time.Minute, "config:set", "--name", "nginx/hosts/public/name", "--value", hosts[i]); err != nil {
					t.Errorf("config:set in %s: %v\n%s", p.name, err, out)
				}
				if out, err := p.tryRun(5*time.Minute, "start"); err != nil {
					t.Errorf("start of %s: %v\n%s", p.name, err, out)
				}
			}()
		}
		wg.Wait()

		conf, err := os.ReadFile(filepath.Join(install.dir, "aruntime", "ctx", "proxy.conf"))
		if err != nil {
			t.Fatal(err)
		}
		for _, host := range hosts {
			if !strings.Contains(string(conf), host) {
				t.Errorf("round %d: %s is not in proxy.conf", round, host)
			}
		}
	}
}
