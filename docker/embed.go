package dockerassets

import "embed"

// Every directory in docker/ has to be named here by hand, and omitting one is
// invisible from source (templates are read off disk) while breaking the
// platform in every built binary. embed_test.go compares this list against the
// directory and fails when they disagree — add the platform in both places.
//
//go:embed all:general all:magento2 all:medusa all:prestashop all:saleor all:shopify all:shopware all:woocommerce all:bigcommerce all:spree all:sylius all:custom all:languages all:snippets
var FS embed.FS
