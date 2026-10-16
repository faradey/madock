package configs

import (
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
)

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stderr
	os.Stderr = w
	fn()
	os.Stderr = saved
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

const otherProjectsXML = `<config><scopes><default><rabbitmq><password>` + fmtc.EncryptedPrefix + `abc123</password></rabbitmq></default></scopes></config>`

// `start` in one project read another project's config, met a ciphertext madock
// Pro had written there, and warned as if it were this project's own.
func TestParsingAnotherProjectsConfigSaysNothing(t *testing.T) {
	savedProvider := secretsProvider
	secretsProvider = nil
	secretReportOnce = sync.Once{}
	t.Cleanup(func() { secretsProvider = savedProvider; secretReportOnce = sync.Once{} })

	var conf map[string]string
	out := captureStderr(t, func() { conf = ParseXmlBytes([]byte(otherProjectsXML)) })
	if out != "" {
		t.Fatalf("parsing a config warned:\n%s", out)
	}
	if !fmtc.IsCiphertext(conf["scopes/default/rabbitmq/password"]) {
		t.Fatalf("the stored value should pass through untouched, got %q", conf["scopes/default/rabbitmq/password"])
	}
}

func TestTheCurrentProjectsCiphertextIsNamed(t *testing.T) {
	savedProvider := secretsProvider
	secretsProvider = nil
	secretReportOnce = sync.Once{}
	t.Cleanup(func() { secretsProvider = savedProvider; secretReportOnce = sync.Once{} })

	conf := map[string]string{
		"rabbitmq/password": fmtc.EncryptedPrefix + "abc123",
		"db/password":       "plain",
	}
	out := captureStderr(t, func() { reportCiphertextOf("shop", conf) })
	if !strings.Contains(out, "project shop") || !strings.Contains(out, "rabbitmq/password") {
		t.Fatalf("warning should name the project and the key:\n%s", out)
	}

	secretReportOnce = sync.Once{}
	if out := captureStderr(t, func() {
		reportCiphertextOf("shop", map[string]string{"db/password": "plain"})
	}); out != "" {
		t.Fatalf("a config with no ciphertext warned:\n%s", out)
	}
}
