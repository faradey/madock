package execute

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"io"
	"strings"
)

// `--json` was declared globally and read by nobody here, so `db:execute --json`
// answered with the client's ordinary TSV. That is not a cosmetic gap: a dump of
// the carrier accounts — the one set of credentials on this machine that nobody
// can reissue — was taken with `db:execute --json "SELECT * FROM
// extmag_shipper_account"` and archived as a `.json` file that is not JSON, with
// one value already corrupted in it.
//
// **The obvious repair is the one to avoid.** Post-processing the client's batch
// output, or asking the server for `JSON_ARRAYAGG`, both fail on the same thing:
// the mysql client in batch mode escapes backslashes, so the `\"` that JSON uses
// to escape a quote arrives as `\\"` and the document no longer parses.
// `TO_BASE64` gets the bytes past the client, but MariaDB breaks base64 with a
// newline every 76 characters, and the aggregate is silently cut off at
// `group_concat_max_len` — 1024 bytes against a dump of 8634 on the stand where
// this was measured. That last one is the worst outcome available: the file looks
// like JSON, parses as JSON, and is missing most of the rows.
//
// So nothing here parses the client's text output. `--xml` is a structured format
// the client produces itself: it needs no knowledge of the columns, so an
// arbitrary `SELECT *` works; NULL arrives as an attribute rather than as a word
// that a string could also spell; and the escaping is XML's, which a real parser
// undoes. The conversion below is between two well-defined formats instead of a
// guess about where one value ends.

// mysqlResultSet is one `<resultset>` of `mysql --xml`. A statement that returns
// nothing produces none, and several statements in one `-e` produce several.
type mysqlResultSet struct {
	Rows []mysqlRow `xml:"row"`
}

type mysqlRow struct {
	Fields []mysqlField `xml:"field"`
}

type mysqlField struct {
	Name string `xml:"name,attr"`
	// Nil is xsi:nil, which is how the client distinguishes NULL from the
	// four-character string "NULL" — the distinction its TSV output loses.
	Nil   string `xml:"nil,attr"`
	Value string `xml:",chardata"`
}

func (f mysqlField) isNull() bool {
	return f.Nil == "true" || f.Nil == "1"
}

// mysqlXMLToJSON converts `mysql --xml` output into a JSON array of objects and
// returns how many result sets it read.
//
// Values are strings, including numeric ones. The client carries no types — the
// column is a name and a body of text — so anything else would be this function
// guessing, and the guess has a wrong answer that matters: a zip code, a version
// string and an account number are all "numbers" that must not lose a leading
// zero or gain a float. NULL is the one value that is not a string, because the
// client does say which those are.
//
// Column order is preserved, so a row reads the way the SELECT was written.
func mysqlXMLToJSON(r io.Reader, w io.Writer) (sets int, err error) {
	decoder := xml.NewDecoder(r)
	out := bufio.NewWriter(w)

	if _, err = out.WriteString("["); err != nil {
		return sets, err
	}

	rows := 0
	for {
		token, tokenErr := decoder.Token()
		if tokenErr == io.EOF {
			break
		}
		if tokenErr != nil {
			return sets, tokenErr
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "resultset" {
			continue
		}
		sets++

		var set mysqlResultSet
		if err = decoder.DecodeElement(&set, &start); err != nil {
			return sets, err
		}

		for _, row := range set.Rows {
			if rows > 0 {
				if _, err = out.WriteString(","); err != nil {
					return sets, err
				}
			}
			if err = writeRow(out, row); err != nil {
				return sets, err
			}
			rows++
		}
	}

	if _, err = out.WriteString("]\n"); err != nil {
		return sets, err
	}

	return sets, out.Flush()
}

func writeRow(out *bufio.Writer, row mysqlRow) error {
	if _, err := out.WriteString("{"); err != nil {
		return err
	}

	for i, field := range row.Fields {
		if i > 0 {
			if _, err := out.WriteString(","); err != nil {
				return err
			}
		}

		name, err := json.Marshal(field.Name)
		if err != nil {
			return err
		}
		if _, err = out.Write(name); err != nil {
			return err
		}
		if _, err = out.WriteString(":"); err != nil {
			return err
		}

		if field.isNull() {
			if _, err = out.WriteString("null"); err != nil {
				return err
			}
			continue
		}

		value, err := json.Marshal(field.Value)
		if err != nil {
			return err
		}
		if _, err = out.Write(value); err != nil {
			return err
		}
	}

	_, err := out.WriteString("}")

	return err
}

// postgresJSONQuery wraps a query so the server does the encoding.
//
// PostgreSQL can turn a row into JSON without being told its columns, which is
// what makes this safe for an arbitrary SELECT. `coalesce` is what keeps an empty
// result an empty array rather than the empty output psql prints for a NULL, and
// the trailing semicolon has to go because the query becomes a subquery.
//
// A statement that returns no rows at all — an UPDATE, a DDL — cannot be wrapped
// and psql says so. That is the right answer: there is no JSON for it, and
// inventing one would be the defect this whole change is about.
func postgresJSONQuery(query string) string {
	trimmed := strings.TrimSpace(query)
	trimmed = strings.TrimRight(trimmed, ";")
	trimmed = strings.TrimSpace(trimmed)

	return "SELECT coalesce(json_agg(row_to_json(t)), '[]'::json) FROM (" + trimmed + ") t"
}
