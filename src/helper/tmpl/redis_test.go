package tmpl

import "testing"

// A value redis would refuse, or one that would break the YAML it is written
// into, stops the render with the setting named. The second column is the
// quote and the newline: written unchecked into `command:` they end the string
// and the list, and compose refuses the whole stack.
func TestRedisSettingsAreCheckedBeforeTheyReachCompose(t *testing.T) {
	for value, ok := range map[string]bool{
		"256mb":         true,
		"1GB":           true,
		"1073741824":    true,
		"512k":          true,
		"256 mb":        false,
		"-1":            false,
		"256mb\"":       false,
		"256mb\n- evil": false,
		"lots":          false,
		"1.5gb":         false,
	} {
		if _, err := RedisMaxmemory(value); (err == nil) != ok {
			t.Errorf("maxmemory %q: err=%v, want accepted=%v", value, err, ok)
		}
	}
	for value, ok := range map[string]bool{
		"allkeys-lru":      true,
		"volatile-ttl":     true,
		"noeviction":       true,
		"allkeys_lru":      false,
		"ALLKEYS-LRU":      false,
		"lru":              false,
		"allkeys-lru\"\n-": false,
	} {
		if _, err := RedisPolicy(value); (err == nil) != ok {
			t.Errorf("maxmemory_policy %q: err=%v, want accepted=%v", value, err, ok)
		}
	}
}
