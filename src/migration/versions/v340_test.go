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
