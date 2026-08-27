package execute

import (
	"encoding/json"
	"strings"
	"testing"
)

// The values here are the ones that broke every earlier attempt, kept together
// because each of them defeated a different approach: the backslash is what the
// mysql client escapes in batch mode, so a JSON document built server-side
// arrived with `\\"` where `\"` was meant and would not parse; the tab and the
// newline are what make the client's TSV ambiguous in the first place; and
// "NULL" as a string is indistinguishable from an actual NULL in that output.
//
// One of these — a value with a single backslash in it, in a `signature` column —
// is already in the recovery archive corrupted, which is why this is a test
// rather than a note.
func TestMysqlXMLToJSONSurvivesTheValuesThatBrokeEveryOtherRoute(t *testing.T) {
	const document = `<?xml version="1.0"?>
<resultset statement="SELECT * FROM extmag_shipper_account" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <row>
	<field name="id">1</field>
	<field name="signature">a\b</field>
	<field name="quoted">he said &quot;yes&quot;</field>
	<field name="tabbed">before	after</field>
	<field name="multiline">first
second</field>
	<field name="empty"></field>
	<field name="spelled_null">NULL</field>
	<field name="really_null" xsi:nil="true" />
	<field name="unicode">Привіт</field>
	<field name="zip">01234</field>
  </row>
</resultset>
`

	var out strings.Builder
	sets, err := mysqlXMLToJSON(strings.NewReader(document), &out)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	if sets != 1 {
		t.Errorf("counted %d result sets, want 1", sets)
	}

	var rows []map[string]any
	if err := json.Unmarshal([]byte(out.String()), &rows); err != nil {
		t.Fatalf("the output is not JSON, which is the whole defect: %v\n%s", err, out.String())
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}

	row := rows[0]
	for field, want := range map[string]any{
		"id":           "1",
		"signature":    `a\b`,
		"quoted":       `he said "yes"`,
		"tabbed":       "before\tafter",
		"multiline":    "first\nsecond",
		"empty":        "",
		"spelled_null": "NULL",
		"really_null":  nil,
		"unicode":      "Привіт",
		// A leading zero is why values stay strings: guessing a type here turns
		// a zip code, a version or an account number into a number that no
		// longer says what it said.
		"zip": "01234",
	} {
		if got, ok := row[field]; !ok {
			t.Errorf("%s is missing from the row", field)
		} else if got != want {
			t.Errorf("%s = %#v, want %#v", field, got, want)
		}
	}
}

// A statement that returns nothing produces no result set at all, and an empty
// array is the answer a consumer can act on. Printing nothing would leave a file
// that no parser accepts — the same failure in a quieter form.
func TestMysqlXMLToJSONAnswersAnEmptyArrayForNoResultSet(t *testing.T) {
	var out strings.Builder
	sets, err := mysqlXMLToJSON(strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	if sets != 0 {
		t.Errorf("counted %d result sets, want 0", sets)
	}
	if out.String() != "[]\n" {
		t.Errorf("got %q, want an empty array", out.String())
	}
}

// Several statements in one -e produce several `<resultset>` elements, which is
// not a single-rooted document — so this has to be read as a stream rather than
// unmarshalled whole. The count is returned so the caller can say the rows were
// merged instead of letting somebody count them and draw a conclusion.
func TestMysqlXMLToJSONReadsEveryResultSet(t *testing.T) {
	const document = `<?xml version="1.0"?>
<resultset statement="SELECT 1 AS one" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <row><field name="one">1</field></row>
</resultset>
<?xml version="1.0"?>
<resultset statement="SELECT 2 AS two" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <row><field name="two">2</field></row>
</resultset>
`

	var out strings.Builder
	sets, err := mysqlXMLToJSON(strings.NewReader(document), &out)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	if sets != 2 {
		t.Errorf("counted %d result sets, want 2", sets)
	}

	var rows []map[string]any
	if err := json.Unmarshal([]byte(out.String()), &rows); err != nil {
		t.Fatalf("the output is not JSON: %v\n%s", err, out.String())
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
}

// Column order is the order of the SELECT. JSON objects carry no order by
// specification, but the file is read by people as well, and a dump whose
// columns shuffle between runs is one nobody can diff.
func TestMysqlXMLToJSONKeepsColumnOrder(t *testing.T) {
	const document = `<?xml version="1.0"?>
<resultset xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <row>
	<field name="zebra">1</field>
	<field name="apple">2</field>
  </row>
</resultset>
`

	var out strings.Builder
	if _, err := mysqlXMLToJSON(strings.NewReader(document), &out); err != nil {
		t.Fatalf("conversion failed: %v", err)
	}

	if !strings.Contains(out.String(), `{"zebra":"1","apple":"2"}`) {
		t.Errorf("the columns were reordered:\n%s", out.String())
	}
}

func TestPostgresJSONQueryWrapsTheQueryAsASubquery(t *testing.T) {
	cases := map[string]string{
		"SELECT * FROM accounts":  `SELECT coalesce(json_agg(row_to_json(t)), '[]'::json) FROM (SELECT * FROM accounts) t`,
		"  SELECT 1 AS one ;  ":   `SELECT coalesce(json_agg(row_to_json(t)), '[]'::json) FROM (SELECT 1 AS one) t`,
		"SELECT 1;;":              `SELECT coalesce(json_agg(row_to_json(t)), '[]'::json) FROM (SELECT 1) t`,
		"SELECT ';' AS semicolon": `SELECT coalesce(json_agg(row_to_json(t)), '[]'::json) FROM (SELECT ';' AS semicolon) t`,
	}

	for query, want := range cases {
		if got := postgresJSONQuery(query); got != want {
			t.Errorf("postgresJSONQuery(%q) =\n  %s\nwant\n  %s", query, got, want)
		}
	}
}
