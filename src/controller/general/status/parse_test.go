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
