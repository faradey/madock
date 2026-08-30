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
	ArgsType interface{}
	Before   []Handler
	After    []Handler

	// Global marks a command that does not belong to a project: it neither reads
	// a project's configuration nor talks to its containers, so it runs anywhere.
	// `setup` counts, because it is what creates a project.
	//
	// Everything else is project-scoped, and the dispatcher refuses to run it in a
	// directory that is not a project. The default is deliberately the strict one:
	// most commands are project-scoped, and the failure of forgetting the flag on
	// a global command is immediate and loud, while the failure of forgetting a
	// check inside a project command is silent. `stop` never had that check, took
	// the directory name for a project name, and drove docker compose with
	// whatever generated file happened to carry that name — leftovers from a
	// version long gone included.
	Global bool

	// PassThrough marks a command whose arguments are not madock's to read: it
	// hands os.Args[2:] to another program — composer, bin/magento, npm, wp-cli,
	// a shell — and `--help` there is a request for that program's help.
	//
	// It exists so the dispatcher can answer `madock <command> --help` itself for
	// everything else. That used to be each command's own job, done by calling
	// attr.Parse, and a command that skipped it ran instead of explaining itself.
	// `install` skipped it: `madock install --help` on an installed project built
	// `bin/magento setup:install` from the project config and ran it, over a live
	// database, having already printed the admin password on the way. Somebody
	// typing --help has asked for exactly the opposite of that.
	//
	// It was not one command. Measured across the registry on 2026-08-20: eight in
	// madock (install, stop, ssl:rebuild, mcp, mftf:init, compress, uncompress,
	// config:cache:clean) and over fifty in madock-pro, backup:create,
	// firewall:setup, server:init and shared-db:unshare among them. So the check
	// belongs in the one place every command passes through, and the default is
	// deliberately the safe one: forgetting this flag on a pass-through command
	// prints madock's help instead of composer's, which is visible and harmless,
	// while forgetting to parse in a command was silent and ran it.
	PassThrough bool

	// JSONOutput marks a command that actually honours --json.
	//
	// The flag is declared once, on the argument struct every command embeds, so
	// every command accepts it and most of them ignore it in silence. That is not
	// a tidiness problem: `db:execute --json "SELECT * FROM
	// extmag_shipper_account"` was accepted, answered with the client's ordinary
	// TSV, and the result was archived as a `.json` file that is not JSON — a dump
	// of the one set of credentials on this machine that nobody can reissue. The
	// command did nothing wrong that could be seen; a flag that is accepted reads
	// as a flag that works.
	//
	// So an unimplemented --json is now refused by the dispatcher, and the default
	// is deliberately the strict one: forgetting this on a command that formats
	// JSON breaks that command loudly the first time anyone runs it, while the
	// other way round is exactly the silence above. Pass-through commands are
	// exempt — there --json belongs to composer, npm or bin/magento, and madock
	// has no business reading it.
	//
	// madock-pro registers its own definitions and marks its own, the way it does
	// for Global through a resolver.
	JSONOutput bool
}

var registry = make(map[string]*Definition)
var globalMiddlewares []Middleware
var scopeResolvers []ScopeResolver

// ScopeResolver answers whether a command belongs to a project, for a layer that
// knows better than madock does. `decided` false means "no opinion".
type ScopeResolver func(def *Definition) (global bool, decided bool)

// AddScopeResolver registers such an answer.
//
// It exists because the Global flag is a property of a definition, and a layer
// built on madock registers definitions madock never sees: madock-pro adds around
// a hundred and ten aliases, and whole families of them — server:*, firewall:*,
// dns:*, disk:*, service:*, the :all group — act on the machine rather than on a
// project. Left to the default they would all be refused outside a project, which
// is where they are actually run.
//
// A resolver keeps the strict default for madock's own commands, where forgetting
// the flag has to fail loudly, while letting the layer that owns the knowledge
// express it as a rule instead of a list of aliases nobody will maintain.
func AddScopeResolver(resolver ScopeResolver) {
	scopeResolvers = append(scopeResolvers, resolver)
}

// IsGlobal reports whether a command may run outside a project. Resolvers are
// asked first, in registration order, and the flag on the definition is the
// fallback.
func IsGlobal(def *Definition) bool {
	for _, resolver := range scopeResolvers {
		if global, decided := resolver(def); decided {
			return global
		}
	}

	return def.Global
}

// Register adds a command definition to the registry
func Register(def *Definition) {
	for _, alias := range def.Aliases {
		registry[alias] = def
	}
}

// Use adds a global middleware applied to all commands.
//
// Extension point for madock-pro: the licence gate is registered here, which is
// what makes a paid command refuse before it runs. Unreachable from this module
// by design.
func Use(m Middleware) {
	globalMiddlewares = append(globalMiddlewares, m)
}

// AddBefore adds a before-hook to the command registered under the given alias.
//
// Extension point for madock-pro, and eleven of its hooks arrive here. See
// AddAfter for why this looks unreachable and is not.
func AddBefore(alias string, hook Handler) {
	if def, ok := registry[alias]; ok {
		def.Before = append(def.Before, hook)
	}
}

// AddAfter adds an after-hook to the command registered under the given alias.
//
// Extension point for madock-pro. Nothing in this repository calls it, and that
// is correct: pro registers 26 after-hooks through it — deploy, backup, cron,
// ssl and the rest — from its own `init`. A reachability analysis of this module
// alone reports it as unreachable, which is how the whole hook mechanism came to
// be listed as dead code in an audit on 2026-08-31.
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
