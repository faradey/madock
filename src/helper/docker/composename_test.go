package docker

import "testing"

// The label and the container name are Compose's, not the registry's: project
// ShopU's containers are madock_shopu-…, labelled madock_shopu. A lookup by
// madock_ShopU found nothing, and project:purge then reported a project
// completely removed with its containers and volumes running — reproduced on a
// VM before the fix.
func TestDockerNamesFollowCompose(t *testing.T) {
	if got := ComposeProjectName("ShopU"); got != "madock_shopu" {
		t.Errorf("ComposeProjectName(ShopU) = %q, want madock_shopu", got)
	}
	if got := ComposeProjectName("shop+1"); got != "madock_shop1" {
		t.Errorf("ComposeProjectName(shop+1) = %q, want madock_shop1", got)
	}
	conf := map[string]string{"container_name_prefix": "madock_", "activeScope": "Staging"}
	if got := GetContainerName(conf, "Shop+1", "php"); got != "madock_shop1-staging-php-1" {
		t.Errorf("GetContainerName = %q, want madock_shop1-staging-php-1", got)
	}
}
