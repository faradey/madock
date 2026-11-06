package lockfile

// Hold has no lock to take on Windows: several madock processes at once at one
// installation are a server's case, and servers are Linux.
func Hold(string, func()) (func(), error) {
	return func() {}, nil
}
