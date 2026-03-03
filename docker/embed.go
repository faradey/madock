package dockerassets

import "embed"

//go:embed all:general all:magento2 all:prestashop all:shopify all:shopware all:custom all:languages all:snippets
var FS embed.FS
