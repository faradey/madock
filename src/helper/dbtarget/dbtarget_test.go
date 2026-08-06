package dbtarget

import "testing"

// resetResolvers clears the package-level resolver list so one test cannot leak
// a resolver into the next.
func resetResolvers(t *testing.T) {
	t.Helper()
	saved := resolvers
	resolvers = nil
	t.Cleanup(func() { resolvers = saved })
}

func baseConf() map[string]string {
	return map[string]string{
		"db/enabled":       "true",
		"db/type":          "mysql",
		"db/repository":    "mariadb",
		"db/version":       "10.6",
		"db/database":      "shop",
		"db/user":          "db",
		"db/password":      "db",
		"db/root_password": "rootpw",
		"db2/enabled":      "false",
	}
}

func TestResolveLocal(t *testing.T) {
	resetResolvers(t)

	target, ok := Resolve(baseConf(), "demo", "db")
	if !ok {
		t.Fatal("Resolve reported no database for a project that runs its own db service")
	}
	if target.Container != "demo-db-1" {
		t.Errorf("Container = %q, want %q", target.Container, "demo-db-1")
	}
	if target.Host != "db" {
		t.Errorf("Host = %q, want %q", target.Host, "db")
	}
	if target.Database != "shop" || target.RootPassword != "rootpw" {
		t.Errorf("credentials = %q/%q, want shop/rootpw", target.Database, target.RootPassword)
	}
	if target.Shared {
		t.Error("Shared is true for a local database")
	}
}

// The db2 service carries its own credentials. Reading them off the db/* keys —
// which is what the commands did before this package existed — sends the client
// at db2 with db's password and db's schema name.
func TestResolveDb2UsesItsOwnCredentials(t *testing.T) {
	resetResolvers(t)

	conf := baseConf()
	conf["db2/enabled"] = "true"
	conf["db2/database"] = "second"
	conf["db2/user"] = "second_user"
	conf["db2/password"] = "second_pw"
	conf["db2/root_password"] = "second_root"

	target, ok := Resolve(conf, "demo", "db2")
	if !ok {
		t.Fatal("Resolve reported no database for an enabled db2 service")
	}
	if target.Database != "second" {
		t.Errorf("Database = %q, want %q", target.Database, "second")
	}
	if target.RootPassword != "second_root" {
		t.Errorf("RootPassword = %q, want %q", target.RootPassword, "second_root")
	}
	if target.Type != "mysql" {
		t.Errorf("Type = %q, want mysql: db2 is always MySQL", target.Type)
	}
}

// Only db and db2 hold a database. Assuming an unknown name exists would build
// a container name nothing ever created, and the user would get Docker's "No
// such container" instead of being told they named a service that is not one.
func TestResolveUnknownServiceIsNotATarget(t *testing.T) {
	resetResolvers(t)

	for _, service := range []string{"shared", "php", "typo"} {
		if _, ok := Resolve(baseConf(), "demo", service); ok {
			t.Errorf("Resolve returned a target for service %q", service)
		}
	}
}

// ...but a registered resolver may still own such a name, and it is asked first.
func TestResolveUnknownServiceCanBeClaimed(t *testing.T) {
	resetResolvers(t)

	Register(func(conf map[string]string, projectName, service string) (Target, bool) {
		if service != "shared" {
			return Target{}, false
		}
		return Target{Container: "provider-db-1", Shared: true}, true
	})

	if _, ok := Resolve(baseConf(), "demo", "shared"); !ok {
		t.Error("a resolver could not claim a service name the local path rejects")
	}
}

func TestResolveDb2DisabledIsNotATarget(t *testing.T) {
	resetResolvers(t)

	if _, ok := Resolve(baseConf(), "demo", "db2"); ok {
		t.Error("Resolve returned a target for a disabled db2 service")
	}
}

// A project with db/enabled=false has no container to exec into. Reporting that
// here is what turns "No such container" into a sentence that names the cause.
func TestResolveNoLocalDatabase(t *testing.T) {
	resetResolvers(t)

	conf := baseConf()
	conf["db/enabled"] = "false"

	if _, ok := Resolve(conf, "demo", "db"); ok {
		t.Error("Resolve returned a target for a project whose db service is switched off")
	}
}

// Every project created before db/enabled existed omits the key, and the compose
// templates render the db service for those. Resolve has to agree.
func TestResolveMissingEnabledKeyMeansEnabled(t *testing.T) {
	resetResolvers(t)

	conf := baseConf()
	delete(conf, "db/enabled")

	if _, ok := Resolve(conf, "demo", "db"); !ok {
		t.Error("Resolve treated an absent db/enabled as disabled")
	}
}

func TestRegisteredResolverWins(t *testing.T) {
	resetResolvers(t)

	Register(func(conf map[string]string, projectName, service string) (Target, bool) {
		return Target{
			Project:   "provider",
			Service:   service,
			Container: "provider-db-1",
			Host:      "127.0.0.1",
			Type:      "mysql",
			Database:  "shared",
			Shared:    true,
		}, true
	})

	// db/enabled=false so the local path could not have produced this.
	conf := baseConf()
	conf["db/enabled"] = "false"

	target, ok := Resolve(conf, "consumer", "db")
	if !ok {
		t.Fatal("a registered resolver did not answer")
	}
	if target.Container != "provider-db-1" || !target.Shared {
		t.Errorf("target = %+v, want the provider's container", target)
	}
	if target.Project != "provider" {
		t.Errorf("Project = %q, want provider", target.Project)
	}
}

func TestDecliningResolverFallsThroughToLocal(t *testing.T) {
	resetResolvers(t)

	Register(func(conf map[string]string, projectName, service string) (Target, bool) {
		return Target{}, false
	})

	target, ok := Resolve(baseConf(), "demo", "db")
	if !ok {
		t.Fatal("Resolve gave up after a resolver declined")
	}
	if target.Container != "demo-db-1" {
		t.Errorf("Container = %q, want the local container", target.Container)
	}
}

func TestClientNames(t *testing.T) {
	cases := []struct {
		repository, version string
		client, dump        string
	}{
		{"mariadb", "10.6", "mariadb", "mariadb-dump"},
		{"mariadb", "10.5", "mariadb", "mariadb-dump"},
		{"mariadb", "10.4", "mysql", "mysqldump"},
		{"mysql", "8.4", "mysql", "mysqldump"},
	}

	for _, c := range cases {
		target := Target{Repository: c.repository, Version: c.version}
		if got := target.MySQLClient(); got != c.client {
			t.Errorf("%s %s: MySQLClient() = %q, want %q", c.repository, c.version, got, c.client)
		}
		if got := target.MySQLDump(); got != c.dump {
			t.Errorf("%s %s: MySQLDump() = %q, want %q", c.repository, c.version, got, c.dump)
		}
	}
}

func TestHasLocalIgnoresResolvers(t *testing.T) {
	resetResolvers(t)

	Register(func(conf map[string]string, projectName, service string) (Target, bool) {
		return Target{Container: "provider-db-1", Shared: true}, true
	})

	conf := baseConf()
	conf["db/enabled"] = "false"

	// A shared database must not make a snapshot think there is a local volume.
	if HasLocal(conf, "db") {
		t.Error("HasLocal is true for a project whose database lives elsewhere")
	}
}
