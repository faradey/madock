package dockerassets

import "embed"

//go:embed all:general all:magento2 all:medusa all:prestashop all:saleor all:shopify all:shopware all:woocommerce all:custom all:languages all:snippets
var FS embed.FS
