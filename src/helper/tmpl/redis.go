package tmpl

import (
	"fmt"
	"regexp"
	"strings"
)

// redisMemory is what redis-server accepts for --maxmemory: a whole number of
// bytes, or one with a unit — k, kb, m, mb, g, gb, any case.
var redisMemory = regexp.MustCompile(`^(?i)[0-9]+(b|k|kb|m|mb|g|gb)?$`)

// redisPolicies are the eviction policies redis-server knows.
var redisPolicies = map[string]bool{
	"noeviction":      true,
	"allkeys-lru":     true,
	"allkeys-lfu":     true,
	"allkeys-random":  true,
	"volatile-lru":    true,
	"volatile-lfu":    true,
	"volatile-random": true,
	"volatile-ttl":    true,
}

// RedisMaxmemory checks redis/maxmemory before it reaches the compose file.
//
// The value is written into the redis service's command, inside the YAML compose
// reads. Unchecked, a value with a quote or a newline would break that file for
// the whole stack, and one redis does not understand would make the container
// exit at start with "Bad directive" — after the stack had been taken down. Here
// the render stops first and says which setting.
func RedisMaxmemory(v any) (string, error) {
	s := strings.TrimSpace(fmt.Sprint(v))
	if !redisMemory.MatchString(s) {
		return "", fmt.Errorf("redis/maxmemory %q is not a size redis accepts: a whole number, optionally with b, k, kb, m, mb, g or gb (256mb, 1gb)", s)
	}
	return s, nil
}

// RedisPolicy checks redis/maxmemory_policy the same way, against the policies
// redis-server knows.
func RedisPolicy(v any) (string, error) {
	s := strings.TrimSpace(fmt.Sprint(v))
	if !redisPolicies[s] {
		return "", fmt.Errorf("redis/maxmemory_policy %q is not one redis knows: noeviction, allkeys-lru, allkeys-lfu, allkeys-random, volatile-lru, volatile-lfu, volatile-random or volatile-ttl", s)
	}
	return s, nil
}
