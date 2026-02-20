package configs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetOriginalGeneralConfig_EmbeddedOnly(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", tmpDir)
	CleanCache()

	conf := GetOriginalGeneralConfig()

	// Embedded defaults should provide these values
	checks := map[string]string{
		"db/root_password": "password",
		"db/password":      "magento",
		"php/version":      "8.2",
		"platform":         "magento2",
		"db/repository":    "mariadb",
		"rabbitmq/version": "3.12.10",
	}

	for key, want := range checks {
		got, ok := conf[key]
		if !ok {
			t.Errorf("key %q missing from embedded defaults", key)
			continue
		}
		if got != want {
			t.Errorf("key %q = %q, want %q", key, got, want)
		}
	}
}

func TestGetOriginalGeneralConfig_MergeFileOverEmbedded(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", tmpDir)
	CleanCache()

	// Write a partial config with only db/root_password overridden
	fileData := map[string]string{
		"db/root_password": "STRONG_PASSWORD_123",
	}
	configPath := filepath.Join(tmpDir, "config.xml")
	SaveInFile(configPath, fileData, "default")
	CleanCache()

	conf := GetOriginalGeneralConfig()

	// File value should win
	if got := conf["db/root_password"]; got != "STRONG_PASSWORD_123" {
		t.Errorf("db/root_password = %q, want %q", got, "STRONG_PASSWORD_123")
	}

	// Embedded should fill the gap for keys not in the file
	if got := conf["php/version"]; got != "8.2" {
		t.Errorf("php/version = %q, want %q (embedded should fill gap)", got, "8.2")
	}
	if got := conf["platform"]; got != "magento2" {
		t.Errorf("platform = %q, want %q (embedded should fill gap)", got, "magento2")
	}
}

func TestGetOriginalGeneralConfig_FileValuesWinOverEmbedded(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", tmpDir)
	CleanCache()

	// Write config where file overrides the embedded default for db/root_password
	fileData := map[string]string{
		"db/root_password": "custom_root_pw",
		"db/password":      "custom_db_pw",
	}
	configPath := filepath.Join(tmpDir, "config.xml")
	SaveInFile(configPath, fileData, "default")
	CleanCache()

	conf := GetOriginalGeneralConfig()

	// File values must win over embedded defaults
	if got := conf["db/root_password"]; got != "custom_root_pw" {
		t.Errorf("db/root_password = %q, want %q (file should win)", got, "custom_root_pw")
	}
	if got := conf["db/password"]; got != "custom_db_pw" {
		t.Errorf("db/password = %q, want %q (file should win)", got, "custom_db_pw")
	}
}

func TestGetOriginalGeneralConfig_EmptyFileValueFilledByEmbedded(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", tmpDir)
	CleanCache()

	// Write config with an empty value for db/root_password.
	// GeneralConfigMapping fills empty values from the main (embedded) config.
	fileData := map[string]string{
		"db/root_password": "",
	}
	configPath := filepath.Join(tmpDir, "config.xml")
	SaveInFile(configPath, fileData, "default")
	CleanCache()

	conf := GetOriginalGeneralConfig()

	// Empty file value should be filled by embedded "password"
	if got := conf["db/root_password"]; got != "password" {
		t.Errorf("db/root_password = %q, want %q (embedded should fill empty)", got, "password")
	}
}

func TestGetOriginalGeneralConfig_NoEmbeddedFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", tmpDir)
	CleanCache()

	// Verify no config.xml exists yet
	configPath := filepath.Join(tmpDir, "config.xml")
	if _, err := os.Stat(configPath); err == nil {
		t.Fatal("config.xml should not exist in temp dir")
	}

	conf := GetOriginalGeneralConfig()

	// Should still return embedded defaults
	if len(conf) == 0 {
		t.Fatal("GetOriginalGeneralConfig() returned empty map with no filesystem config")
	}
}
