//go:build e2e

package e2e

import (
	"encoding/json"
	"slices"
	"testing"
	"time"
)

// TestARunningProjectIsNotConfusedWithItsNamesake stops a project while another
// one whose name starts the same way goes on, as a template and its clone do.
func TestARunningProjectIsNotConfusedWithItsNamesake(t *testing.T) {
	install := newInstallation(t)
	shop := install.project("e2eshop")
	clone := install.project("e2eshop-2")
	for _, p := range []*project{shop, clone} {
		p.run(setupTimeout, "setup", "-y", "--platform=custom", "--language=none", "--hosts="+p.name+".test")
		p.run(20*time.Minute, "start")
	}

	running := func() []string {
		t.Helper()
		var rows []struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal([]byte(shop.run(time.Minute, "project:list", "--running", "--json")), &rows); err != nil {
			t.Fatal(err)
		}
		names := make([]string, 0, len(rows))
		for _, row := range rows {
			names = append(names, row.Name)
		}
		return names
	}

	if names := running(); !slices.Contains(names, "e2eshop") || !slices.Contains(names, "e2eshop-2") {
		t.Errorf("both started, running: %v", names)
	}
	shop.run(5*time.Minute, "stop")
	if names := running(); slices.Contains(names, "e2eshop") || !slices.Contains(names, "e2eshop-2") {
		t.Errorf("e2eshop stopped, running: %v", names)
	}
}
