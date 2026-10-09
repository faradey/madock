package sqldump

import (
	"io"
	"strings"
	"testing"
)

func strip(t *testing.T, in string) string {
	t.Helper()
	out, err := io.ReadAll(StripGtidPurged(strings.NewReader(in)))
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// The statement as mysqldump 8 and 9 write it, spanning two lines, between the
// lines a restore needs.
func TestTheGtidStatementIsDroppedAndNothingElse(t *testing.T) {
	dump := "SET @@SESSION.SQL_LOG_BIN= 0;\n" +
		"SET @@GLOBAL.GTID_PURGED=/*!80000 '+'*/ '3E11FA47-71CA-11E1-9E33-C80AA9429562:1-5,\n" +
		"4E11FA47-71CA-11E1-9E33-C80AA9429562:1-3';\n" +
		"CREATE TABLE `kept` (`note` varchar(32));\n" +
		"INSERT INTO `kept` VALUES ('before-the-backup');\n"

	got := strip(t, dump)

	if strings.Contains(got, "GTID_PURGED") || strings.Contains(got, "4E11FA47") {
		t.Errorf("the GTID statement survived:\n%s", got)
	}
	for _, want := range []string{"SQL_LOG_BIN", "CREATE TABLE `kept`", "'before-the-backup'"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q was dropped with it:\n%s", want, got)
		}
	}
}

// A row whose data mentions the statement is data. The filter used to match the
// text anywhere in a line, and would have dropped this INSERT from the restore.
func TestARowMentioningTheStatementIsKept(t *testing.T) {
	row := "INSERT INTO `notes` VALUES ('run SET @@GLOBAL.GTID_PURGED by hand');\n"

	if got := strip(t, row); got != row {
		t.Errorf("a data row was changed:\n got %q\nwant %q", got, row)
	}
}

func TestADumpWithoutGtidsPassesUnchanged(t *testing.T) {
	dump := "CREATE TABLE `t` (`id` int);\nINSERT INTO `t` VALUES (1),(2);\n"

	if got := strip(t, dump); got != dump {
		t.Errorf("got %q, want %q", got, dump)
	}
}
