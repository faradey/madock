package platform

func init() {
	// PHP-based platforms with full chown and cron support
	Register("magento2", &BaseHandler{
		MainContainer: "php",
		ChownDirs:     []string{"workdir", "/var/www/.composer", "/var/www/.npm"},
		HasCron:       true,
	})

	Register("shopware", &BaseHandler{
		MainContainer: "php",
		ChownDirs:     []string{"workdir", "/var/www/.composer", "/var/www/.npm"},
		HasCron:       true,
	})

	Register("prestashop", &BaseHandler{
		MainContainer: "php",
		ChownDirs:     []string{"workdir", "/var/www/.composer", "/var/www/.npm"},
		HasCron:       true,
	})

	// PHP-based platforms without .npm directory
	Register("shopify", &BaseHandler{
		MainContainer: "php",
		ChownDirs:     []string{"workdir", "/var/www/.composer"},
		HasCron:       true,
	})

	Register("custom", &BaseHandler{
		MainContainer: "php",
		ChownDirs:     []string{"workdir", "/var/www/.composer"},
		HasCron:       true,
	})

}
