package versions

import (
	"os"
	"regexp"
	"strings"
)

// ReadMagentoEnvDbCreds reads the default DB connection (username, password,
// dbname) from a Magento 2 app/etc/env.php. The file is a PHP array, but the
// default connection is the first one declared inside db -> connection, so the
// scan is narrowed to that block before reading the positional keys — no php
// runtime is required. ok is false when the file is missing/unreadable or the
// credentials cannot be located.
func ReadMagentoEnvDbCreds(envPath string) (user, password, dbname string, ok bool) {
	data, err := os.ReadFile(envPath)
	if err != nil {
		return "", "", "", false
	}
	content := string(data)

	// Walk down 'db' -> 'connection' -> 'default' so the cache/session 'default'
	// blocks and the 'indexer' connection cannot shadow the real credentials.
	for _, anchor := range []string{"'db'", "'connection'", "'default'"} {
		if i := strings.Index(content, anchor); i != -1 {
			content = content[i+len(anchor):]
		}
	}

	keyVal := func(key string) string {
		m := regexp.MustCompile(`['"]` + key + `['"]\s*=>\s*['"]([^'"]*)['"]`).FindStringSubmatch(content)
		if len(m) == 2 {
			return m[1]
		}
		return ""
	}

	dbname = keyVal("dbname")
	user = keyVal("username")
	password = keyVal("password")
	if dbname == "" || user == "" {
		return "", "", "", false
	}
	return user, password, dbname, true
}
