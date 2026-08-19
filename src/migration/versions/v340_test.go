package versions

import (
	"os"
	"testing"

	"github.com/faradey/madock/v3/src/helper/configs"
)

func TestMigrateDbType_AddsMySQL(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "db-type-migration-*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <activeScope>default</activeScope>
    <scopes>
        <default>
            <db>
                <repository>mariadb</repository>
                <version>10.6</version>
            </db>
        </default>
    </scopes>
</config>`

	tmpFile.WriteString(xmlContent)
	tmpFile.Close()

	projectConf := map[string]string{
		"db/repository": "mariadb",
	}

	migrateDbType(tmpPath, projectConf)

	result := configs.ParseXmlFile(tmpPath)
	if result["scopes/default/db/type"] != "mysql" {
		t.Errorf("db/type = %q, want %q", result["scopes/default/db/type"], "mysql")
	}
}

func TestMigrateDbType_AddsPostgreSQL(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "db-type-migration-*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <activeScope>default</activeScope>
    <scopes>
        <default>
            <db>
                <repository>postgres</repository>
                <version>16</version>
            </db>
        </default>
    </scopes>
</config>`

	tmpFile.WriteString(xmlContent)
	tmpFile.Close()

	projectConf := map[string]string{
		"db/repository": "postgres",
	}

	migrateDbType(tmpPath, projectConf)

	result := configs.ParseXmlFile(tmpPath)
	if result["scopes/default/db/type"] != "postgresql" {
		t.Errorf("db/type = %q, want %q", result["scopes/default/db/type"], "postgresql")
	}
}

func TestMigrateDbType_AddsMongoDB(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "db-type-migration-*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <activeScope>default</activeScope>
    <scopes>
        <default>
            <db>
                <repository>mongo</repository>
                <version>7</version>
            </db>
        </default>
    </scopes>
</config>`

	tmpFile.WriteString(xmlContent)
	tmpFile.Close()

	projectConf := map[string]string{
		"db/repository": "mongo",
	}

	migrateDbType(tmpPath, projectConf)

	result := configs.ParseXmlFile(tmpPath)
	if result["scopes/default/db/type"] != "mongodb" {
		t.Errorf("db/type = %q, want %q", result["scopes/default/db/type"], "mongodb")
	}
}

func TestMigrateDbType_SkipsIfAlreadySet(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "db-type-migration-*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <activeScope>default</activeScope>
    <scopes>
        <default>
            <db>
                <type>postgresql</type>
                <repository>mariadb</repository>
            </db>
        </default>
    </scopes>
</config>`

	tmpFile.WriteString(xmlContent)
	tmpFile.Close()

	projectConf := map[string]string{
		"db/type":       "postgresql",
		"db/repository": "mariadb",
	}

	migrateDbType(tmpPath, projectConf)

	result := configs.ParseXmlFile(tmpPath)
	// Should keep existing value, not override with detected "mysql"
	if result["scopes/default/db/type"] != "postgresql" {
		t.Errorf("db/type = %q, want %q", result["scopes/default/db/type"], "postgresql")
	}
}

// A project that says it has no database does not get an engine written into
// its config.
//
// `GetDbType` answers "mysql" for anything it cannot read as postgres or mongo,
// nothing at all included, so this wrote a declaration the author never made —
// into the project's own committed file, where a later reader believes it.
// Measured on a project with db/enabled=false and no repository.
func TestMigrateDbType_LeavesADisabledDatabaseAlone(t *testing.T) {
	tmpPath := writeConfig(t, `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <activeScope>default</activeScope>
    <scopes>
        <default>
            <db>
                <enabled>false</enabled>
            </db>
        </default>
    </scopes>
</config>`)

	migrateDbType(tmpPath, map[string]string{"db/enabled": "false"})

	result := configs.ParseXmlFile(tmpPath)
	if got, ok := result["scopes/default/db/type"]; ok {
		t.Errorf("db/type = %q was invented for a project with no database", got)
	}
}

// Nothing to carry across is not a reason to write a default.
//
// The key is a convenience: `GetDbType` falls back to the repository — and to
// "mysql" — at read time either way, so a project whose config never named a
// repository loses nothing by this staying unwritten, and keeps a config that
// claims only what it was given.
func TestMigrateDbType_WritesNothingWithNoRepository(t *testing.T) {
	tmpPath := writeConfig(t, `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <activeScope>default</activeScope>
    <scopes>
        <default>
            <php>
                <version>8.3</version>
            </php>
        </default>
    </scopes>
</config>`)

	migrateDbType(tmpPath, map[string]string{"php/version": "8.3"})

	result := configs.ParseXmlFile(tmpPath)
	if got, ok := result["scopes/default/db/type"]; ok {
		t.Errorf("db/type = %q was invented for a project that named no repository", got)
	}
}

// writeConfig puts an XML config in a temporary file and returns its path.
func writeConfig(t *testing.T, content string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "db-type-migration-*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}
	return tmpFile.Name()
}

func TestMigrateDbType_DefaultsToMySQL(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "db-type-migration-*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <activeScope>default</activeScope>
    <scopes>
        <default>
            <db>
                <repository>mysql</repository>
            </db>
        </default>
    </scopes>
</config>`

	tmpFile.WriteString(xmlContent)
	tmpFile.Close()

	projectConf := map[string]string{
		"db/repository": "mysql",
	}

	migrateDbType(tmpPath, projectConf)

	result := configs.ParseXmlFile(tmpPath)
	if result["scopes/default/db/type"] != "mysql" {
		t.Errorf("db/type = %q, want %q", result["scopes/default/db/type"], "mysql")
	}
}
