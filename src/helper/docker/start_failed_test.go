package docker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStartFailed(t *testing.T) {
	tests := []struct {
		name      string
		dead      []ServiceState
		proxyDown bool
		want      bool
	}{
		{name: "all up", want: false},
		{name: "the proxy is down", proxyDown: true, want: true},
		{name: "a service died", dead: []ServiceState{{Service: "db", State: "exited", ExitCode: 137}}, want: true},
		{name: "a one-shot finished", dead: []ServiceState{{Service: "init", State: "exited"}}, want: false},
		{name: "a service never started", dead: []ServiceState{{Service: "db", State: "created"}}, want: true},
		{name: "a one-shot finished and a service died", dead: []ServiceState{
			{Service: "init", State: "exited"},
			{Service: "db", State: "dead"},
		}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StartFailed(tt.dead, tt.proxyDown); got != tt.want {
				t.Errorf("StartFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemovingAMarkerAnotherProcessRemovedIsNotAFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conf-cache")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := removeIfThere(path); err != nil {
		t.Errorf("present: %v", err)
	}
	if err := removeIfThere(path); err != nil {
		t.Errorf("already gone: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("still there: %v", err)
	}
}
