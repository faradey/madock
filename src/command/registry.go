package command

// Handler is a function that executes a command
type Handler func()

// Middleware wraps a Handler to add cross-cutting behavior (auth, logging, etc.)
type Middleware func(Handler) Handler

// Definition defines a command with its aliases and handler
type Definition struct {
	Aliases  []string
	Handler  Handler
	Help     string
	Category string
	Before   []Handler
	After    []Handler
}

var registry = make(map[string]*Definition)
var globalMiddlewares []Middleware

// Register adds a command definition to the registry
func Register(def *Definition) {
	for _, alias := range def.Aliases {
		registry[alias] = def
	}
}

// Use adds a global middleware applied to all commands
func Use(m Middleware) {
	globalMiddlewares = append(globalMiddlewares, m)
}

// AddBefore adds a before-hook to the command registered under the given alias
func AddBefore(alias string, hook Handler) {
	if def, ok := registry[alias]; ok {
		def.Before = append(def.Before, hook)
	}
}

// AddAfter adds an after-hook to the command registered under the given alias
func AddAfter(alias string, hook Handler) {
	if def, ok := registry[alias]; ok {
		def.After = append(def.After, hook)
	}
}

// Get returns a copy of the command definition with middleware chain applied
func Get(name string) (*Definition, bool) {
	def, ok := registry[name]
	if !ok {
		return nil, false
	}

	if len(globalMiddlewares) == 0 && len(def.Before) == 0 && len(def.After) == 0 {
		return def, true
	}

	// Build wrapped handler: before hooks → original handler → after hooks
	original := def.Handler
	wrapped := func() {
		for _, hook := range def.Before {
			hook()
		}
		original()
		for _, hook := range def.After {
			hook()
		}
	}

	// Apply global middlewares (first registered = outermost)
	for i := len(globalMiddlewares) - 1; i >= 0; i-- {
		wrapped = globalMiddlewares[i](wrapped)
	}

	// Return a copy so the original definition stays untouched
	copy := *def
	copy.Handler = wrapped
	return &copy, true
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
