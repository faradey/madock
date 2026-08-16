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
