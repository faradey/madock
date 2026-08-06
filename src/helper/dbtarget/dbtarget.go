// Package dbtarget answers one question for every database command: which
// container to run the client in, and with which host and credentials.
//
// Each db:* command used to build that itself — container name from the current
// project, credentials from the current project's config. That is only correct
// while the database lives inside the project's own stack. A project whose
// database is owned by another project has no db container at all, and those
// commands failed with a bare "No such container".
//
// Resolve is the single place that decides, and Register lets a hook decide
// instead. The hook that knows about shared databases lives outside this
// repository, so nothing here needs to know it exists.
package dbtarget

import (
	"strings"

	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
)

// Target describes the database a command should operate on.
type Target struct {
	// Project owns the database. Equal to the current project unless the
	// database is shared, in which case it is the providing project.
	Project string
	// Service is the compose service holding the database ("db" or "db2").
	Service string
	// Container is the container the client binary runs in. For a shared
	// database that is the provider's container, reached over the proxy network.
	Container string
	// Host is the value passed to the client's -h flag from inside Container.
	Host string
	// Type is mysql, postgresql or mongodb.
	Type string
	// Database, User and Password address the schema being worked on.
	Database, User, Password string
	// RootPassword is the MySQL/MariaDB superuser password on the server owning
	// the schema. The db:* commands use it so that export can lock tables and
	// import can create them regardless of what the application account may do.
	RootPassword string
	// Repository and Version identify the database image, which decides whether
	// the client binaries are named mysql* or mariadb*.
	Repository, Version string
	// Shared reports that the database belongs to another project.
	Shared bool
}

// Resolver reports the target for a service, or false to let the next resolver
// (and ultimately the local project) answer.
type Resolver func(projectConf map[string]string, projectName, service string) (Target, bool)

var resolvers []Resolver

// Register adds a resolver. Resolvers are asked in registration order, before
// the local project is consulted.
func Register(r Resolver) {
	if r != nil {
		resolvers = append(resolvers, r)
	}
}

// Resolve returns the target for the given service. ok is false when there is
// nothing to talk to: no resolver claimed the project and its own service is
// switched off.
func Resolve(projectConf map[string]string, projectName, service string) (Target, bool) {
	if service == "" {
		service = "db"
	}

	for _, r := range resolvers {
		if target, ok := r(projectConf, projectName, service); ok {
			return target, true
		}
	}

	if !localEnabled(projectConf, service) {
		return Target{}, false
	}

	return Local(projectConf, projectName, service), true
}

// MustResolve is Resolve with the failure spelled out instead of left to
// surface as a missing container further down.
func MustResolve(projectConf map[string]string, projectName, service string) Target {
	target, ok := Resolve(projectConf, projectName, service)
	if !ok {
		logger.Fatalln("Project \"" + projectName + "\" has no \"" + service + "\" service: " + service +
			"/enabled is false and no shared database is configured for it.")
	}
	return target
}

// Local builds the target for a database inside the project's own stack.
func Local(projectConf map[string]string, projectName, service string) Target {
	// GetDbType only describes the primary service; db2 is always MySQL.
	dbType := "mysql"
	if service == "db" {
		dbType = configs.GetDbType(projectConf)
	}

	return Target{
		Project:      projectName,
		Service:      service,
		Container:    docker.GetContainerName(projectConf, projectName, service),
		Host:         service,
		Type:         dbType,
		Database:     projectConf[service+"/database"],
		User:         projectConf[service+"/user"],
		Password:     projectConf[service+"/password"],
		RootPassword: projectConf[service+"/root_password"],
		// db2 is built from the same image as db (db.Dockerfile), so the client
		// binaries are named the same on both.
		Repository: projectConf["db/repository"],
		Version:    projectConf["db/version"],
	}
}

// MySQLClient returns the interactive client binary name. MariaDB renamed its
// binaries in 10.5 and keeps the mysql* names only as deprecated symlinks.
func (t Target) MySQLClient() string {
	return t.clientName("mariadb", "mysql")
}

// MySQLDump returns the dump binary name, by the same rule as MySQLClient.
func (t Target) MySQLDump() string {
	return t.clientName("mariadb-dump", "mysqldump")
}

func (t Target) clientName(mariaName, mysqlName string) string {
	if strings.ToLower(t.Repository) == "mariadb" && configs.CompareVersions(t.Version, "10.5") != -1 {
		return mariaName
	}
	return mysqlName
}

// Origin describes where the database lives, for command output.
func (t Target) Origin() string {
	if t.Shared {
		return "shared database of project \"" + t.Project + "\" (" + t.Service + ")"
	}
	return "local " + t.Service
}

// HasLocal reports whether the project runs the database service in its own
// stack. Commands that work on the container or its data volume rather than on
// the schema — snapshots, for one — must ask this instead of Resolve: a shared
// database belongs to its provider, and copying or replacing its files from
// here would take every other consumer with it.
func HasLocal(projectConf map[string]string, service string) bool {
	return localEnabled(projectConf, service)
}

// localEnabled reports whether the project runs the service itself. An absent
// db/enabled means enabled: that is how the compose templates treat an
// unsubstituted placeholder, and every project created before the flag existed
// relies on it.
func localEnabled(projectConf map[string]string, service string) bool {
	switch service {
	case "db":
		return projectConf["db/enabled"] != "false"
	case "db2":
		return projectConf["db2/enabled"] == "true"
	default:
		// A service this package does not know about is assumed to be running.
		return true
	}
}
