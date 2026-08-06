package configs

// ResolveMainService names the compose service that runs the application code.
//
// It lives here because three packages need the answer and none of them may
// import the others: the compose generator, the platform handlers, and the
// docker helpers that chown the working directory after an up. Two copies of
// this switch already existed, and the one that did not get updated is how
// `--with-chown` ended up reaching into a php container on a project that runs
// none.
//
// fallback is returned for "php" and for an unset language, so a caller that
// knows its platform's main container can pass it.
func ResolveMainService(projectConf map[string]string, fallback string) string {
	if lang, ok := projectConf["language"]; ok && lang != "" && lang != "php" {
		switch lang {
		case "nodejs":
			return "nodejs"
		case "python":
			return "python"
		case "golang":
			return "golang"
		case "ruby":
			return "ruby"
		case "none":
			return "app"
		}
	}
	return fallback
}
