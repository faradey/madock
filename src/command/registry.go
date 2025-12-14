package command

// Handler is a function that executes a command
type Handler func()

// Definition defines a command with its aliases and handler
type Definition struct {
	Aliases []string
	Handler Handler
	Help    string
}

var registry = make(map[string]*Definition)

// Register adds a command definition to the registry
func Register(def *Definition) {
	for _, alias := range def.Aliases {
		registry[alias] = def
	}
}

// Get returns the command definition for the given name
func Get(name string) (*Definition, bool) {
	def, ok := registry[name]
	return def, ok
}

// GetAll returns all unique command definitions
func GetAll() []*Definition {
	seen := make(map[*Definition]bool)
	var result []*Definition
	for _, def := range registry {
		if !seen[def] {
			seen[def] = true
			result = append(result, def)
		}
	}
	return result
}
