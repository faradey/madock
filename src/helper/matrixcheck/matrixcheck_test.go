package matrixcheck

import (
	"strings"
	"testing"
)

func magento247(over map[string]string) map[string]string {
	conf := map[string]string{
		"platform":                  "magento2",
		"language":                  "php",
		"php/version":               "8.3",
		"db/enabled":                "true",
		"db/type":                   "mysql",
		"db/version":                "10.6",
		"search/opensearch/enabled": "true",
		"search/opensearch/version": "2.12.1",
		"redis/enabled":             "false",
		"redis/version":             "9.9",
	}
	for k, v := range over {
		conf[k] = v
	}
	return conf
}

// Magento 2.4.7 supports PHP 8.3 at most (Adobe's requirements, the matrix's
// line). 8.4 is past it and is named; the matrix's own line and anything older
// are not, since the matrix has no floor to hold them against.
func TestAVersionPastTheLineIsNamed(t *testing.T) {
	got := Outside(magento247(map[string]string{"php/version": "8.4"}), "2.4.7")
	if len(got) != 1 || !strings.Contains(got[0], "php/version 8.4 is newer than magento2 2.4.7 supports (8.3") {
		t.Fatalf("got %q", got)
	}

	for _, php := range []string{"8.3", "8.2"} {
		if got := Outside(magento247(map[string]string{"php/version": php}), "2.4.7"); len(got) != 0 {
			t.Errorf("php %s: %q", php, got)
		}
	}
}

// A patch above the matrix's line is on that line: OpenSearch 2.12.1 for
// Adobe's 2.12.0. A disabled service is not checked at all — redis 9.9 above.
func TestOnlyTheLineCountsAndOnlyWhatRuns(t *testing.T) {
	if got := Outside(magento247(nil), "2.4.7"); len(got) != 0 {
		t.Fatalf("got %q", got)
	}
	got := Outside(magento247(map[string]string{"redis/enabled": "true"}), "2.4.7")
	if len(got) != 1 || !strings.HasPrefix(got[0], "redis/version 9.9") {
		t.Errorf("an enabled redis past the line: %q", got)
	}
}

// The matrix's db line is for its own engine: MySQL 8.4 is not read against
// MariaDB 10.6.
func TestTheDatabaseIsComparedOnlyWithItsOwnEngine(t *testing.T) {
	mysql := magento247(map[string]string{"db/repository": "mysql", "db/version": "99.0"})
	if got := Outside(mysql, "2.4.7"); len(got) != 0 {
		t.Errorf("a MySQL version was held against the MariaDB line: %q", got)
	}
	if got := Outside(magento247(map[string]string{"db/version": "11.4"}), "2.4.7"); len(got) != 1 {
		t.Errorf("MariaDB 11.4 on 2.4.7: %q", got)
	}
}

// OpenSearch on a Magento older than 2.4.3-p2 is NotCompatible in the matrix.
func TestATooolTheMatrixCallsNotCompatibleIsNamed(t *testing.T) {
	got := Outside(magento247(nil), "2.4.2")
	found := false
	for _, line := range got {
		if strings.Contains(line, "opensearch does not run with magento2 2.4.2") {
			found = true
		}
	}
	if !found {
		t.Errorf("got %q", got)
	}
}

// Medusa's matrix answers db with an image, "postgres:17". That is no version,
// and read as one it sorts below everything, so any configured db would be
// "newer".
func TestAnAnswerThatIsNoVersionIsNotCompared(t *testing.T) {
	conf := map[string]string{
		"platform": "medusa", "language": "nodejs",
		"db/enabled": "true", "db/type": "postgresql", "db/version": "17",
	}
	if got := Outside(conf, "2"); len(got) != 0 {
		t.Errorf("got %q", got)
	}
}

func TestAPlatformWithoutAMatrixSaysNothing(t *testing.T) {
	if got := Outside(map[string]string{"platform": "no-such", "php/version": "99"}, "1.0"); got != nil {
		t.Errorf("got %q", got)
	}
}
