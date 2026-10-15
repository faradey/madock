package setup

import (
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/model/versions/magento2"
)

// "NotCompatible" used to become the OpenSearch image tag. A release the
// matrix says OpenSearch cannot run is refused before that can happen.
func TestOpenSearchIsRefusedWhereTheMatrixSaysNotCompatible(t *testing.T) {
	old := magento2.GetVersions("2.4.2")
	if got := openSearchRefusal(old, ""); !strings.Contains(got, "2.4.3-p2") {
		t.Errorf("Magento 2.4.2 with OpenSearch should be refused and say from which release it works, got %q", got)
	}

	if got := openSearchRefusal(old, "1.2.0"); got != "" {
		t.Errorf("an explicit --search-engine-version is taken at its word, got refusal %q", got)
	}

	for _, v := range []string{"2.4.3-p2", "2.4.6", "2.3.7-p4"} {
		if got := openSearchRefusal(magento2.GetVersions(v), ""); got != "" {
			t.Errorf("Magento %s supports OpenSearch, got refusal %q", v, got)
		}
	}
}
