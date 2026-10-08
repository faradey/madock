package project

import (
	"reflect"
	"testing"

	"github.com/faradey/madock/v4/src/helper/testenv"
)

// The case found on this machine: a project's own compose file, in the old
// syntax, reading the port arithmetic madock dropped years ago. It rendered
// ":80" without a word.
func TestAProjectTemplateReadingARetiredKeyIsNamed(t *testing.T) {
	testenv.SetupWith(t, "unknownkeys", "unknownkeys.test", map[string]string{
		"worker/programs/queue/command": "php bin/queue",
	})
	r := newRenderer("unknownkeys", "", nil)

	legacy := `ports:
  - "{{{nginx/port/project}}}:80"
  - "{{{nginx/port/project_ssl}}}:443"
memory: {{{php/limits/memory}}}
`
	if got, want := unknownKeysIn("docker-compose.yml", legacy, r.Values, r.Data), []string{"nginx/port/project", "nginx/port/project_ssl"}; !reflect.DeepEqual(got, want) {
		t.Errorf("old syntax: unknown keys %v, want %v", got, want)
	}

	current := `memory: {{{.php.limits.memory}}}
{{{range $host := .nginx.hosts}}}{{{$host.name}}}{{{end}}}
{{{range $name, $p := .worker.programs}}}{{{$name}}}{{{end}}}
setup: {{{.php.limits.max_execution_time_web}}}
newer: {{{.php.packages.someday}}}
`
	if got, want := unknownKeysIn("default.conf", current, r.Values, r.Data), []string{"php/packages/someday"}; !reflect.DeepEqual(got, want) {
		t.Errorf("current syntax: unknown keys %v, want %v — a setting, the host list, a named-children block and a renamed key are all answered", got, want)
	}
}
