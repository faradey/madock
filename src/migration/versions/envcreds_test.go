package versions

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "env.php")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestReadMagentoEnvDbCreds(t *testing.T) {
	env := `<?php
return [
    'backend' => ['frontName' => 'admin'],
    'cache' => ['frontend' => ['default' => ['id_prefix' => 'x']]],
    'db' => [
        'table_prefix' => '',
        'connection' => [
            'default' => [
                'host' => 'db',
                'dbname' => 'db',
                'username' => 'db',
                'password' => 'S9fggJYCiOVykHwABxtNVWxD',
                'active' => '1',
            ],
            'indexer' => [
                'host' => 'wrong',
                'dbname' => 'wrong',
                'username' => 'wrong',
                'password' => 'wrong',
            ],
        ],
    ],
];`
	user, password, dbname, ok := ReadMagentoEnvDbCreds(writeTemp(t, env))
	if !ok {
		t.Fatal("expected ok=true")
	}
	if user != "db" || dbname != "db" || password != "S9fggJYCiOVykHwABxtNVWxD" {
		t.Fatalf("got user=%q dbname=%q password=%q", user, dbname, password)
	}
}

func TestReadMagentoEnvDbCredsMissingFile(t *testing.T) {
	if _, _, _, ok := ReadMagentoEnvDbCreds("/no/such/env.php"); ok {
		t.Fatal("expected ok=false for missing file")
	}
}

func TestReadMagentoEnvDbCredsNoCreds(t *testing.T) {
	if _, _, _, ok := ReadMagentoEnvDbCreds(writeTemp(t, "<?php return ['x' => 1];")); ok {
		t.Fatal("expected ok=false when creds absent")
	}
}
