// Package tmpl renders madock's docker templates with text/template from the
// standard library.
//
// It replaces an engine written by hand — three passes over the text, one for
// {{{include}}}, one for {{{key}}} substitution and one for <<<if>>> — whose
// weaknesses were not syntactic. A condition there was a substring search for
// "false", so a template could not compare two values and Go grew fake config
// keys to stand in for the comparisons it could not express (db/type_is_mysql,
// db/use_default_auth_plugin). There were no loops, so the indentation of a
// compose file was baked into a strings.Join in Go source. And an error had no
// address: one unbalanced tag made the engine abandon a file with every
// conditional unresolved and write the result out anyway.
//
// text/template answers all of that at no cost in dependencies. What it needs
// from us is three things the standard library cannot know:
//
//  1. Delimiters. They stay {{{ }}} — the obvious alternative, <<< >>>, is a
//     bash here-string, and three Dockerfile templates legitimately contain one
//     (`IFS='.' read major minor patch <<< "{{{php/version}}}"`).
//  2. A data tree. Config keys are slash-separated strings, so php/xdebug/enabled
//     has to become .php.xdebug.enabled, and the string "false" has to become a
//     real bool — every non-empty string is true in a Go template, so a config
//     value left as a string would make every {{{if}}} fire.
//  3. Tolerance for a key the project does not have. A shared snippet asks about
//     memcached/enabled on a platform whose config has never heard of memcached;
//     the old engine left the placeholder standing and read it as false. Making
//     that a render failure would break `madock start` on every such project, so
//     instead every key a template mentions is seeded into the tree as an empty
//     value before execution — absent means false, as it always did. Typos are
//     caught where they belong, by a test over the whole tree of templates, not
//     by a fatal error on a user's machine.
package tmpl

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"text/template/parse"
)

// The delimiters. Kept as constants because the converter, the tests and the
// key audit all have to agree with the renderer about what an action looks like.
const (
	LeftDelim  = "{{{"
	RightDelim = "}}}"
)

// Renderer holds everything one project's templates are rendered against.
//
// It deliberately depends on nothing inside madock: ports are allocated through
// a function the caller supplies, and snippets are read through another. That
// keeps this package a leaf — it can be imported by configs, by project and by
// the shared proxy generator without an import cycle, and its tests need no
// madock installation on disk.
type Renderer struct {
	// Values is the flat, slash-keyed configuration, exactly as
	// configs.GetProjectConfig returns it, plus whatever the caller computes
	// (project_name, scope, main_service, os/user/uid …).
	Values map[string]string

	// Data carries what a configuration file cannot hold: an ordered list. It
	// is keyed the same way and applied on top of Values, so nginx/hosts
	// replaces the map the nginx/hosts/<code>/name keys would have built with a
	// slice a template can range over in order and index into.
	//
	// This is where the joined strings went. The old engine had no loops, so
	// hosts were joined in Go with the YAML indentation of a compose file baked
	// into the separator (strings.Join(onlyHosts, "\n      ")). The indentation
	// belongs in the template that has it.
	Data map[string]any

	// Snippet reads an include by its template name, which is the path used in
	// the template: "snippets/docker-compose/php.yml". The caller owns the
	// override chain — a project's own .madock/docker/ wins over the embedded
	// copy — so it lives with the caller and not here.
	Snippet func(name string) (string, error)

	// Port returns the host port published for a service. It is a function and
	// not a value because resolving one has a side effect: it allocates the
	// port, writing it into the project's port registry. Templates call it as
	// {{{port "livereload"}}}, which is honest about that in a way {{{port/livereload}}}
	// never was.
	Port func(service string) (int, error)

	// OnLegacy is called when a template turns out to be written in the old
	// <<<if>>> syntax, before it is converted and rendered. Only a project's own
	// overrides under .madock/docker/ can be, and they keep working — but
	// silently would mean nobody ever updates them.
	OnLegacy func(name string, notes []string)
}

// Render parses body under the given name, resolves every {{{template}}} it
// reaches, and executes it.
//
// name is used in error messages, so it should be the path the template came
// from rather than something invented at the call site.
func (r *Renderer) Render(name, body string) (string, error) {
	root := template.New(name).Delims(LeftDelim, RightDelim).Option("missingkey=error").Funcs(r.funcMap())

	if _, err := root.Parse(r.source(name, body)); err != nil {
		return "", err
	}
	if err := r.loadSnippets(root); err != nil {
		return "", err
	}

	data, err := r.data(root)
	if err != nil {
		return "", err
	}

	var out strings.Builder
	if err := root.Execute(&out, data); err != nil {
		return "", err
	}

	return out.String(), nil
}

// source is every template's way in, the root and each snippet alike.
//
// A template written against the old syntax is converted here rather than read
// by a second engine kept alive for it. Only a project's own overrides under
// .madock/docker/ can still be in that shape — everything madock ships was
// converted once, in the tree — and an override can be either the file being
// rendered or a snippet it pulls in, so both go through here.
func (r *Renderer) source(name, body string) string {
	if !IsLegacy(body) {
		return body
	}

	converted, notes := Legacy(body)
	if r.OnLegacy != nil {
		r.OnLegacy(name, notes)
	}
	return converted
}

// StubFuncs is the function vocabulary with the right names and arities and no
// behaviour, for anything that parses a template without rendering it: the
// converter, the test that every embedded template parses, and the audit of the
// keys they read.
func StubFuncs() template.FuncMap {
	return (&Renderer{}).funcMap()
}

// Keys lists every configuration key a template reads, in the slash form the
// configuration uses — .php.xdebug.enabled comes back as php/xdebug/enabled.
//
// It is what replaces missingkey=error. A key the project does not carry has to
// stay falsy at render time, because a shared snippet asks about memcached on
// platforms whose config has never heard of it; so a typo cannot be caught
// there. It is caught here instead, over the whole tree at once, and for every
// platform rather than only the one somebody happened to start.
//
// Chains rooted at a variable — $host.name inside a range — are not keys and do
// not appear.
func Keys(name, body string) ([]string, error) {
	root, err := template.New(name).Delims(LeftDelim, RightDelim).Funcs(StubFuncs()).Parse(body)
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, path := range referencedFields(root) {
		keys = append(keys, strings.Join(path, "/"))
	}
	return keys, nil
}

// loadSnippets parses every template the set refers to but does not yet define,
// repeating until the set is closed.
//
// This is the include pass, and it is the one place the rewrite is strictly
// better rather than merely different. The old pass was a regex looped "while a
// match remains", with no cycle detection at all: a snippet that included itself
// spun forever. Here a name is loaded once and the loop ends when nothing new
// appears, so a cycle terminates — and if the templates really are recursive,
// text/template stops execution at its own depth limit with a named error
// instead of a hung process.
func (r *Renderer) loadSnippets(root *template.Template) error {
	loaded := map[string]bool{}

	for {
		var missing []string
		for _, name := range referencedTemplates(root) {
			if loaded[name] || root.Lookup(name) != nil {
				continue
			}
			missing = append(missing, name)
		}
		if len(missing) == 0 {
			return nil
		}

		sort.Strings(missing) // deterministic order, so an error names the same snippet on every run
		for _, name := range missing {
			if r.Snippet == nil {
				return fmt.Errorf("template %q includes %q but no snippet loader was configured", root.Name(), name)
			}
			body, err := r.Snippet(name)
			if err != nil {
				return fmt.Errorf("including %q: %w", name, err)
			}
			if _, err := root.New(name).Parse(r.source(name, body)); err != nil {
				return err
			}
			loaded[name] = true
		}
	}
}

// referencedTemplates lists every name reached by a {{{template}}} action in
// any tree currently in the set.
func referencedTemplates(root *template.Template) []string {
	var names []string
	for _, t := range root.Templates() {
		if t.Tree == nil || t.Tree.Root == nil {
			continue
		}
		walk(t.Tree.Root, func(n parse.Node) {
			if tn, ok := n.(*parse.TemplateNode); ok {
				names = append(names, tn.Name)
			}
		})
	}
	return names
}

// data builds the tree the templates are executed against.
//
// Two steps, and the second is what keeps a render from failing on a key the
// project has never had: after the configuration is turned into a tree, every
// field chain the parsed templates mention is seeded with an empty value if the
// tree has nothing at that path. A missing key is therefore falsy and prints as
// nothing, which is what the old engine did by accident and what the templates
// were written against.
func (r *Renderer) data(root *template.Template) (map[string]any, error) {
	tree, err := buildTree(r.Values)
	if err != nil {
		return nil, err
	}

	for _, key := range sortedKeys(r.Data) {
		if err := place(tree, strings.Split(key, "/"), r.Data[key]); err != nil {
			return nil, err
		}
	}

	for _, path := range referencedFields(root) {
		seed(tree, path)
	}

	return tree, nil
}

func sortedKeys(data map[string]any) []string {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// place writes a value at a path, replacing whatever the configuration keys put
// there. Only Data uses it, and replacing is the point: nginx/hosts arrives as
// a map built out of nginx/hosts/<code>/name and leaves as an ordered slice.
func place(tree map[string]any, path []string, value any) error {
	node := tree
	for i, part := range path[:len(path)-1] {
		switch existing := node[part].(type) {
		case map[string]any:
			node = existing
		case nil:
			child := map[string]any{}
			node[part] = child
			node = child
		default:
			return fmt.Errorf("cannot place %q: %q is a value, not a group", strings.Join(path, "/"), strings.Join(path[:i+1], "/"))
		}
	}
	node[path[len(path)-1]] = value
	return nil
}

// referencedFields lists every field chain rooted at dot — .php.xdebug.enabled
// arrives as ["php","xdebug","enabled"].
//
// Chains rooted at a variable ($host.name inside a range) are not field nodes
// and never appear here, which is exactly right: they resolve against the range
// element, not against the configuration.
func referencedFields(root *template.Template) [][]string {
	var fields [][]string
	for _, t := range root.Templates() {
		if t.Tree == nil || t.Tree.Root == nil {
			continue
		}
		walk(t.Tree.Root, func(n parse.Node) {
			switch node := n.(type) {
			case *parse.FieldNode:
				fields = append(fields, node.Ident)
			case *parse.ChainNode:
				// {{{(index .a "b").c}}} and friends. Only the chain that
				// starts at dot tells us anything about the configuration.
				if _, ok := node.Node.(*parse.DotNode); ok {
					fields = append(fields, node.Field)
				}
			}
		})
	}
	return fields
}

// walk visits every node of a parse tree, including the pipelines hidden inside
// control structures. text/template exposes no walker of its own.
func walk(n parse.Node, fn func(parse.Node)) {
	if n == nil {
		return
	}
	fn(n)

	switch node := n.(type) {
	case *parse.ListNode:
		if node == nil {
			return
		}
		for _, child := range node.Nodes {
			walk(child, fn)
		}
	case *parse.ActionNode:
		walk(node.Pipe, fn)
	case *parse.PipeNode:
		if node == nil {
			return
		}
		for _, cmd := range node.Cmds {
			walk(cmd, fn)
		}
	case *parse.CommandNode:
		for _, arg := range node.Args {
			walk(arg, fn)
		}
	case *parse.IfNode:
		walkBranch(&node.BranchNode, fn)
	case *parse.RangeNode:
		walkBranch(&node.BranchNode, fn)
	case *parse.WithNode:
		walkBranch(&node.BranchNode, fn)
	case *parse.TemplateNode:
		walk(node.Pipe, fn)
	case *parse.ChainNode:
		walk(node.Node, fn)
	}
}

func walkBranch(b *parse.BranchNode, fn func(parse.Node)) {
	walk(b.Pipe, fn)
	if b.List != nil {
		walk(b.List, fn)
	}
	if b.ElseList != nil {
		walk(b.ElseList, fn)
	}
}

// buildTree turns slash-keyed configuration into the nested maps a template
// indexes with a dot.
//
// The one conversion in the codebase, on purpose. Two representations of the
// same key is how main_service ended up substituted by hand in one place and by
// the general pass in another, with an eight-line comment explaining why the
// order mattered.
func buildTree(values map[string]string) (map[string]any, error) {
	root := map[string]any{}

	// Sorted, so a conflict between two keys is reported the same way on every
	// run rather than depending on map iteration order.
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		parts := strings.Split(key, "/")
		node := root
		for i, part := range parts[:len(parts)-1] {
			switch existing := node[part].(type) {
			case map[string]any:
				node = existing
			case nil:
				child := map[string]any{}
				node[part] = child
				node = child
			default:
				return nil, fmt.Errorf("config key %q cannot be read as a tree: %q is already a value", key, strings.Join(parts[:i+1], "/"))
			}
		}

		leaf := parts[len(parts)-1]
		if _, taken := node[leaf].(map[string]any); taken {
			return nil, fmt.Errorf("config key %q is both a value and a group of keys", key)
		}
		node[leaf] = coerce(values[key])
	}

	return root, nil
}

// coerce turns the two strings that mean a boolean into real booleans.
//
// This is the one silent regression the rewrite made available. Config values
// are strings; every non-empty string is true in a Go template, so leaving
// "false" alone would make {{{if .php.enabled}}} fire on a disabled service and
// nothing would say a word. Only the exact lowercase words convert — a version
// or a path that happens to contain them stays a string, which is the bug the
// old substring test had.
func coerce(value string) any {
	switch value {
	case "true":
		return true
	case "false":
		return false
	}
	return value
}

// seed writes an empty value at path when the tree has nothing there, so a key
// the project does not carry reads as false instead of failing the render.
//
// It never overwrites, and it stops rather than replacing a value that is
// already a leaf: a template that asks for .db.type.x when db/type is a string
// has a real bug, and execution reporting it by name is the right outcome.
func seed(tree map[string]any, path []string) {
	node := tree
	for _, part := range path[:len(path)-1] {
		switch existing := node[part].(type) {
		case map[string]any:
			node = existing
		case nil:
			child := map[string]any{}
			node[part] = child
			node = child
		default:
			return
		}
	}

	leaf := path[len(path)-1]
	if _, exists := node[leaf]; !exists {
		node[leaf] = ""
	}
}

// funcMap is the whole vocabulary a template has beyond the standard builtins.
//
// Kept short deliberately. Everything here exists because a template cannot
// express it and Go had to leak the answer into the configuration instead —
// version comparison arrived as db/use_default_auth_plugin and db/type_is_mysql,
// and a port arrived as a placeholder whose resolution allocated it.
func (r *Renderer) funcMap() template.FuncMap {
	return template.FuncMap{
		"port": func(service string) (int, error) {
			if r.Port == nil {
				return 0, fmt.Errorf("port %q is asked for by a template that is not rendered with a port allocator", service)
			}
			return r.Port(service)
		},

		"versionGte": func(a, b string) bool { return CompareVersions(a, b) >= 0 },
		"versionGt":  func(a, b string) bool { return CompareVersions(a, b) > 0 },
		"versionLte": func(a, b string) bool { return CompareVersions(a, b) <= 0 },
		"versionLt":  func(a, b string) bool { return CompareVersions(a, b) < 0 },

		// join takes the separator first, the way every other template library
		// does, so that it reads left to right in a pipeline.
		"join": func(sep string, items any) (string, error) {
			list, err := strList(items)
			if err != nil {
				return "", err
			}
			return strings.Join(list, sep), nil
		},

		"lower": strings.ToLower,
		"upper": strings.ToUpper,

		// One memory budget, divided by whoever is reading it. Each engine
		// wants a different setting in a different unit, and a template that
		// cannot do arithmetic leaves only one way to change any of them:
		// copying the file into a project and letting the copy drift.
		"memShare":   memShare,
		"memShareGB": memShareGB,
	}
}

func strList(items any) ([]string, error) {
	switch v := items.(type) {
	case []string:
		return v, nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprint(item))
		}
		return out, nil
	}
	return nil, fmt.Errorf("join: %T is not a list", items)
}

// CompareVersions compares two dotted version strings: 1 if a > b, -1 if a < b,
// 0 if equal. A missing segment counts as zero, so "8.4" and "8.4.0" are equal.
//
// It is a copy of configs.CompareVersions rather than a call to it, and that is
// the price of this package importing nothing from madock. Twelve lines against
// an import cycle through configs, which is where the old engine lives.
func CompareVersions(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	length := len(partsA)
	if len(partsB) > length {
		length = len(partsB)
	}

	for i := 0; i < length; i++ {
		var numA, numB int
		if i < len(partsA) {
			numA, _ = strconv.Atoi(partsA[i])
		}
		if i < len(partsB) {
			numB, _ = strconv.Atoi(partsB[i])
		}
		if numA > numB {
			return 1
		}
		if numA < numB {
			return -1
		}
	}

	return 0
}
