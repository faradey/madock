package configs

import "testing"

func TestDbTypeFromRepository(t *testing.T) {
	cases := map[string]string{
		"postgres":        "postgresql",
		"postgresql":      "postgresql",
		"postgis/postgis": "mysql", // not a postgres image by name; the prefix is what is read
		"mongo":           "mongodb",
		"mongodb":         "mongodb",
		"mysql":           "mysql",
		"mariadb":         "mysql",
		"":                "mysql",
	}

	for repo, want := range cases {
		if got := DbTypeFromRepository(repo); got != want {
			t.Errorf("DbTypeFromRepository(%q) = %q, want %q", repo, got, want)
		}
	}
}

// The trap this exists for: `--db postgres:16` names a repository, and the
// engine written beside it used to come from the interactive question — which
// under --yes nobody answered, leaving MariaDB. The project then had
// db/repository=postgres with db/type=mysql, and the generated compose put the
// postgres image into the MySQL service block with MYSQL_* variables on it.
func TestGetDbType_ExplicitTypeStillWins(t *testing.T) {
	conf := map[string]string{"db/type": "postgresql", "db/repository": "mysql"}

	if got := GetDbType(conf); got != "postgresql" {
		t.Fatalf("an explicit db/type must win over the repository, got %q", got)
	}
}

func TestGetDbType_FallsBackToTheRepository(t *testing.T) {
	conf := map[string]string{"db/repository": "postgres"}

	if got := GetDbType(conf); got != "postgresql" {
		t.Fatalf("with no db/type the repository decides, got %q", got)
	}
}

// The trap: db/type names an engine *family*, and every compose snippet gates on
// one of exactly three values. "mariadb" is a repository wearing the family's
// name, and it matched none of them — so the project came out with no `db`
// service at all, while `start` reported success, `status` listed php and nginx
// without mentioning a database, and `info` printed "type MARIADB, host db".
//
// Found on a project whose config madock itself wrote in March 2024. The
// normalization already existed and was reached only when db/type was empty
// (TestDbTypeFromRepository has "mariadb": "mysql"), which is to say the one
// path that needed it was the one path that skipped it.
func TestGetDbType_NormalizesTheExplicitValue(t *testing.T) {
	cases := map[string]string{
		"mariadb":    "mysql",
		"MariaDB":    "mysql", // as madock's own older configs spell it
		"  MARIADB ": "mysql",
		"percona":    "mysql",
		"mysql":      "mysql",
		"MySQL":      "mysql",
		"postgres":   "postgresql",
		"PostgreSQL": "postgresql",
		"pgsql":      "postgresql",
		"mongo":      "mongodb",
		"MongoDB":    "mongodb",
	}

	for value, want := range cases {
		conf := map[string]string{"db/type": value}
		if got := GetDbType(conf); got != want {
			t.Errorf("db/type=%q gave %q, want %q", value, got, want)
		}
	}
}

// An unreadable value falls back to the repository rather than defaulting
// blindly, so a typo next to a repository that says what it is still gets the
// right engine. What must not happen is the old answer — one that matches no
// snippet, because that one is silent.
func TestGetDbType_UnknownValueDefersToTheRepository(t *testing.T) {
	conf := map[string]string{"db/type": "postgress", "db/repository": "postgres"}
	if got := GetDbType(conf); got != "postgresql" {
		t.Fatalf("an unreadable db/type must defer to the repository, got %q", got)
	}

	conf = map[string]string{"db/type": "nonsense"}
	if got := GetDbType(conf); got != "mysql" {
		t.Fatalf("with nothing else to read the answer is the default, got %q", got)
	}
}

// Every answer has to be one a compose snippet gates on. This is the property
// the bug violated, stated directly: docker/snippets/docker-compose/db.yml tests
// `eq .db.type "mysql"`, db-postgresql.yml "postgresql", db-mongodb.yml
// "mongodb", and there is no fourth file.
func TestGetDbType_AlwaysAnswersAFamilyASnippetGatesOn(t *testing.T) {
	families := map[string]bool{"mysql": true, "postgresql": true, "mongodb": true}

	values := []string{
		"", "mysql", "mariadb", "MariaDB", "percona", "postgres", "postgresql",
		"PostgreSQL", "pgsql", "mongo", "mongodb", "MongoDB", "nonsense", "   ",
	}
	repositories := []string{"", "mariadb", "mysql", "postgres", "mongo", "registry.example.com/team/db"}

	for _, value := range values {
		for _, repository := range repositories {
			conf := map[string]string{"db/type": value, "db/repository": repository}
			if got := GetDbType(conf); !families[got] {
				t.Errorf("db/type=%q db/repository=%q gave %q, which no snippet gates on", value, repository, got)
			}
		}
	}
}
