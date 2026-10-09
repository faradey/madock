// Package sqldump holds what madock does to a SQL dump on its way into a server.
package sqldump

import (
	"bufio"
	"bytes"
	"io"
)

// StripGtidPurged returns a reader that drops mysqldump's
// `SET @@GLOBAL.GTID_PURGED=...;` statement, however many lines it spans.
//
// mysqldump writes it whenever the source server runs with GTIDs, and a server
// whose own GTID_EXECUTED is not empty refuses it with error 3546 — which on a
// restore into the same server is every time, because that server has executed
// those transactions itself. The statement only matters for seeding a replica;
// for putting data back it is the one line that stops the whole import.
//
// Matched at the start of a line, not anywhere in it: a data row that happens to
// contain the text is an INSERT, and dropping it would lose data silently.
func StripGtidPurged(r io.Reader) io.Reader {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		scanner := bufio.NewScanner(r)
		// Allow very large lines (mysqldump extended INSERTs can be huge).
		scanner.Buffer(make([]byte, 1024*1024), 256*1024*1024)

		skipUntilSemicolon := false
		for scanner.Scan() {
			line := scanner.Bytes()
			if skipUntilSemicolon {
				if endsStatement(line) {
					skipUntilSemicolon = false
				}
				continue
			}
			if bytes.HasPrefix(bytes.TrimLeft(line, " \t"), []byte("SET @@GLOBAL.GTID_PURGED")) {
				if !endsStatement(line) {
					skipUntilSemicolon = true
				}
				continue
			}
			if _, err := pw.Write(append(line, '\n')); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
		if err := scanner.Err(); err != nil {
			pw.CloseWithError(err)
		}
	}()
	return pr
}

func endsStatement(line []byte) bool {
	return bytes.HasSuffix(bytes.TrimRight(line, " \t\r"), []byte(";"))
}
