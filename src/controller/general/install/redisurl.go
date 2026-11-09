package install

// RedisURLRewriter may change the address of the project's own Redis before
// `install` writes it into a platform's `.env` (Medusa, Saleor, Spree). It
// gets the project configuration and the address madock built, and returns
// the address to write.
//
// Extension point for madock-pro, which registers one to put the Redis
// password into the address. The `.env` is written in the same shell chain
// that installs and migrates the platform, so a hook after `install` would
// come too late: the platform has already connected by then.
type RedisURLRewriter func(projectConf map[string]string, address string) string

var redisURLRewriters []RedisURLRewriter

// RegisterRedisURLRewriter adds a rewriter. Rewriters run in the order they
// were registered, each on the previous one's result.
func RegisterRedisURLRewriter(r RedisURLRewriter) {
	redisURLRewriters = append(redisURLRewriters, r)
}

// projectRedisURL returns the address of the project's Redis, logical
// database db ("" for none), as the registered rewriters leave it. With none
// registered it is the address madock has always written.
func projectRedisURL(projectConf map[string]string, db string) string {
	address := "redis://redisdb:6379"
	if db != "" {
		address += "/" + db
	}
	for _, r := range redisURLRewriters {
		address = r(projectConf, address)
	}
	return address
}
