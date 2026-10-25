// Package credentials keeps a snapshot and the passwords inside it together.
//
// A database or broker takes its passwords from the configuration once, when
// its volume is first initialised; after that the volume holds them and the
// configuration is only what madock hands to everything that connects. A
// snapshot carries the volume, so it carries the passwords that were current
// when it was taken. Restoring it — or cloning from it — under a configuration
// whose passwords have changed since gives a database that answers "Access
// denied" to every client madock starts, with nothing anywhere saying why.
//
// So a snapshot records the passwords it was taken with, and whatever puts its
// volumes back puts those passwords back into the project's configuration.
package credentials

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/faradey/madock/v4/src/helper/configs"
)

// File is the record's name inside a snapshot directory.
const File = "credentials.xml"

// Keys are the passwords a service reads only when its volume is initialised:
// MySQL and MariaDB (MYSQL_ROOT_PASSWORD, MYSQL_PASSWORD), PostgreSQL and
// MongoDB (db/password is their superuser's), RabbitMQ (RABBITMQ_DEFAULT_PASS).
// Redis, Valkey and the search engines are not here: the templates start them
// without authentication, so their keys reach no service.
var Keys = []string{
	"db/root_password",
	"db/password",
	"db2/root_password",
	"db2/password",
	"rabbitmq/password",
}

// Record writes the passwords conf holds into dir. Written through the config
// writer, so an edition that encrypts secret keys encrypts these too.
func Record(dir string, conf map[string]string) {
	values := map[string]string{}
	for _, key := range Keys {
		if v := conf[key]; v != "" {
			values[key] = v
		}
	}
	if len(values) == 0 {
		return
	}
	configs.SaveInFile(filepath.Join(dir, File), values, "default")
}

// Recorded returns the passwords a snapshot directory was taken with, and
// false for a snapshot made before snapshots recorded them.
func Recorded(dir string) (map[string]string, bool) {
	file := filepath.Join(dir, File)
	if _, err := os.Stat(file); err != nil {
		return nil, false
	}
	values := map[string]string{}
	for key, value := range configs.ParseXmlFile(file) {
		values[strings.TrimPrefix(key, "scopes/default/")] = value
	}
	return values, true
}

// Apply puts the passwords recorded in dir into the project's configuration
// where they differ from conf, and returns the keys it changed. A snapshot
// without a record changes nothing: it was taken before any password could be
// recorded, and the configuration's are the best answer there is.
func Apply(projectName, scope, dir string, conf map[string]string) []string {
	recorded, ok := Recorded(dir)
	if !ok {
		return nil
	}
	if scope == "" {
		scope = "default"
	}
	var changed []string
	for _, key := range Keys {
		v, ok := recorded[key]
		if !ok || v == "" || conf[key] == v {
			continue
		}
		configs.SetParam(projectName, key, v, scope, configs.MadockLevelConfigCode)
		changed = append(changed, key)
	}
	return changed
}
