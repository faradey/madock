package status

import (
	"encoding/json"
	"testing"
)

// One service is the case that was broken, and it is not an edge: a proxy with
// mailpit disabled has exactly one, and so does any project cut down to a single
// container. `docker compose ps --format json` prints one object per line, so
// the count decides the shape of the output — and the old code only wrapped it
// into an array when it could see a `}{` boundary.
//
// The consequence was worse than a missing feature. The decode failed, the error
// was discarded, and `status` answered "No services found" about a proxy that
// was serving every site on the machine.
func TestParseJsonHandlesAnyNumberOfServices(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "nothing running",
			input: "",
			want:  0,
		},
		{
			name:  "one service",
			input: `{"Name":"aruntime-nginx-1","Service":"nginx","State":"running"}`,
			want:  1,
		},
		{
			name: "two services, newline separated",
			input: `{"Name":"aruntime-nginx-1","Service":"nginx","State":"running"}
{"Name":"aruntime-mailcatcher-1","Service":"mailcatcher","State":"running"}`,
			want: 2,
		},
		{
			name:  "already an array",
			input: `[{"Name":"a","Service":"a","State":"running"},{"Name":"b","Service":"b","State":"exited"}]`,
			want:  2,
		},
		{
			// Docker talking rather than answering. The caller now reads stdout
			// alone, which keeps this on stderr where compose puts it — but
			// every `madock status` on a project whose compose file still
			// carried the obsolete top-level `version` key used to print
			// "Could not read the container status: invalid character 'i' in
			// literal true (expecting 'r')", a JSON parser complaining about
			// English. Ignoring a line of prose costs a line of prose; parsing
			// one costs the whole status.
			name: "a warning mixed in with the data",
			input: `WARN[0000] /path/docker-compose.yml: the attribute ` + "`version`" + ` is obsolete, it will be ignored
{"Name":"project-php-1","Service":"php","State":"running"}`,
			want: 1,
		},
		{
			name:  "nothing but a warning",
			input: `WARN[0000] the attribute ` + "`version`" + ` is obsolete, it will be ignored`,
			want:  0,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var decoded []InfoStruct
			if err := json.Unmarshal(parseJson([]byte(testCase.input)), &decoded); err != nil {
				t.Fatalf("the output does not decode as a list: %v\ninput: %s", err, testCase.input)
			}
			if len(decoded) != testCase.want {
				t.Errorf("got %d services, want %d", len(decoded), testCase.want)
			}
		})
	}
}
