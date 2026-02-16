package languages

// GetDefaultVersions returns the default tool versions for a given language
func GetDefaultVersions(language string) map[string]string {
	switch language {
	case "python":
		return map[string]string{
			"python/version": "3.12",
		}
	case "golang":
		return map[string]string{
			"go/version": "1.22",
		}
	case "ruby":
		return map[string]string{
			"ruby/version": "3.3",
		}
	case "nodejs":
		return map[string]string{
			"nodejs/version": "20.19.0",
			"nodejs/yarn/version": "3.6.4",
		}
	}
	return nil
}
