package project

import (
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/testenv"
)

// The default is pinned by the golden fixtures, which render my.cnf; this is
// the other half — a project that asks for durable commits gets them.
func TestFlushLogAtTrxCommitIsASetting(t *testing.T) {
	env := testenv.SetupWith(t, "flushlog", "flushlog.test", map[string]string{
		"db/flush_log_at_trx_commit": "1",
	})

	MakeConf("flushlog")

	cnf := readGenerated(t, env, "ctx/my.cnf")
	if !strings.Contains(cnf, "innodb_flush_log_at_trx_commit = 1\n") {
		t.Errorf("db/flush_log_at_trx_commit did not reach my.cnf:\n%s", cnf)
	}
}
