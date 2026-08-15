package tmpl

import (
	"fmt"
	"regexp"
	"strings"
)

// This file converts madock's old template syntax — the hand-written <<<if>>>
// DSL and its {{{key/with/slashes}}} placeholders — into text/template.
//
// It is not a migration script that ran once. It stays for two reasons:
//
//  1. A template is overridable per project: GetDockerConfigFile looks in
//     <PROJECT>/.madock/docker/ before the embedded copy, and those files belong
//     to whoever wrote them. Converting one at render time, with a warning, is
//     what keeps `madock start` working on a machine whose override was written
//     against the old syntax. The alternative was keeping the old engine alive
//     beside the new one, which is two engines to reason about for the same
//     files.
//  2. The rewrite touched every template there is, so any branch that changes a
//     template conflicts with it wholesale. With the conversion in the tree, a
//     rebase is "take theirs, run tools/tmplconvert, run the golden tests"
//     instead of 276 conflicts.
//
// Nothing here decides anything. Every rule is mechanical, and the keys that
// were never configuration — the booleans Go computed because a template could
// not compare two values — are listed by name in derivedKeys, so a key this
// does not recognise is reported rather than guessed at.

var (
	tagRe         = regexp.MustCompile(`<<<if[^>]*>>>|<<<else>>>|<<<endif>>>`)
	placeholderRe = regexp.MustCompile(`\{\{\{([^{}]+)\}\}\}`)

	// A placeholder in the old syntax is a bare config key: letters, digits,
	// underscores and slashes, and nothing else. Every action in the new syntax
	// starts with something this cannot match — a dot, a trim marker, a
	// variable, a parenthesis — or is a keyword followed by a space.
	legacyPlaceholderRe = regexp.MustCompile(`\{\{\{\s*(include\s+[^{}]+|[A-Za-z][A-Za-z0-9_]*(?:/[A-Za-z0-9_]+)*)\s*\}\}\}`)

	// closingTag is what the converter emits for <<<endif>>>, and the shape a
	// whole-file conditional ends with.
	closingTag = LeftDelim + "- end" + RightDelim

	// The two keywords that stand alone and would otherwise read as a bare key.
	standaloneActions = map[string]bool{"end": true, "else": true, "break": true, "continue": true}
)

// IsLegacy reports whether a template is written in the old syntax.
//
// One conditional tag is proof on its own. Otherwise it takes a placeholder
// that is a bare config key, since that is the one shape the new syntax cannot
// produce.
func IsLegacy(body string) bool {
	if strings.Contains(body, "<<<if") {
		return true
	}
	for _, match := range legacyPlaceholderRe.FindAllStringSubmatch(body, -1) {
		if !standaloneActions[strings.TrimSpace(match[1])] {
			return true
		}
	}
	return false
}

// Legacy rewrites one template and reports anything a reader should look at.
//
// A template already in the new syntax is returned untouched, so running the
// converter over a tree twice is a no-op — which matters, because that is
// exactly what a rebase does: take theirs, run it again, run the golden tests.
func Legacy(body string) (string, []string) {
	if !IsLegacy(body) {
		return body, nil
	}

	var out []string
	var notes []string
	stack := []string{}

	for _, line := range strings.Split(body, "\n") {
		out = append(out, convertLine(line, &stack, &notes)...)
	}

	if len(stack) != 0 {
		notes = append(notes, fmt.Sprintf("%d <<<if without an <<<endif>>>", len(stack)))
	}

	converted := strings.Join(out, "\n")

	// A snippet that is one conditional from top to bottom — which is what
	// nearly every compose snippet is — must render to nothing at all when it
	// is switched off. The newline that follows its last {{{- end}}} is outside
	// the conditional, so it survives, and a default project ended up with
	// eighteen blank lines where its unused services would have been.
	//
	// Dropping it costs the blank line between two services that are both on.
	// The old engine's separation was accidental anyway — it came from the line
	// the include tag sat on — and a compose file that is one service after
	// another reads no worse than one with a blank line every third line and
	// eighteen in a row at the end.
	if converted != body && strings.HasSuffix(strings.TrimRight(converted, "\n"), closingTag) {
		if isWholeFileConditional(converted) {
			converted = strings.TrimRight(converted, "\n")
		} else if !strings.HasSuffix(converted, "\n") {
			// The other side of the same coin: a file that opens with content
			// and closes with a conditional — networks.yml — used to end
			// without a newline harmlessly, because the include was a splice
			// and the newline came from the line the tag sat on. Its last line
			// is now a trim marker, which eats the newline before it, and the
			// compose file ended mid-line on `external: true`.
			converted += "\n"
		}
	}

	return converted, notes
}

// isWholeFileConditional answers whether a converted template is one
// conditional from its first line to its last, which is what nearly every
// compose snippet is. A file that opens with content — networks.yml, volumes.yml
// — is not one, and keeps the newline that ends it.
func isWholeFileConditional(converted string) bool {
	first, _, _ := strings.Cut(converted, "\n")
	return strings.HasPrefix(strings.TrimSpace(first), LeftDelim+"- if ")
}

func convertLine(line string, stack *[]string, notes *[]string) []string {
	if block, ok := loopBlock(line); ok {
		return block
	}
	if include, ok := indentedInclude(line); ok {
		return []string{include}
	}

	tags := tagRe.FindAllStringIndex(line, -1)
	if len(tags) == 0 {
		return []string{substitute(line, notes)}
	}

	var out []string
	buf := ""
	pos := 0
	lastIndent := ""
	afterTag := false

	for _, tag := range tags {
		buf += line[pos:tag[0]]
		text := line[tag[0]:tag[1]]
		rest := line[tag[1]:]

		// Where the tag's own line is indented to. It is cosmetic — the trim
		// markers delete it — but a template nobody can read is a template
		// nobody will edit, so it follows whatever the tag was wrapping.
		indent := ""
		switch {
		case strings.TrimSpace(buf) == "" && buf != "":
			indent = buf
		case leadingSpace(rest) != "":
			indent = leadingSpace(rest)
		}

		if strings.TrimSpace(buf) != "" {
			out = append(out, substitute(indented(buf, afterTag, lastIndent), notes))
		}
		buf = ""
		afterTag = true

		switch {
		case strings.HasPrefix(text, "<<<if"):
			out = append(out, indent+"{{{- if "+condition(text[len("<<<if"):len(text)-len(">>>")], notes)+"}}}")
			*stack = append(*stack, indent)
			lastIndent = indent
		case text == "<<<else>>>":
			indent = top(*stack)
			out = append(out, indent+"{{{- else}}}")
			lastIndent = indent
		case text == "<<<endif>>>":
			indent = top(*stack)
			if len(*stack) > 0 {
				*stack = (*stack)[:len(*stack)-1]
			} else {
				*notes = append(*notes, "an <<<endif>>> with no <<<if")
			}
			out = append(out, indent+"{{{- end}}}")
			lastIndent = indent
		}

		pos = tag[1]
	}

	buf += line[pos:]
	if buf != "" {
		out = append(out, substitute(indented(buf, afterTag, lastIndent), notes))
	}

	return out
}

// indented gives content that shared a line with a tag the indentation the tag
// was standing in for.
//
// `  <<<if{{{db2/enabled}}}>>>dbdata2:<<<endif>>>` is one line holding three
// things, and the volume entry's two spaces are in front of the tag rather than
// in front of the entry. Splitting it into three lines without this puts
// dbdata2 at column zero, which compose reads as a different mapping
// altogether — a whole volumes block silently misplaced.
func indented(text string, afterTag bool, indent string) string {
	if afterTag && leadingSpace(text) == "" {
		return indent + text
	}
	return text
}

// loopBlock converts the two placeholders that were never a value: a list,
// joined in Go with the indentation of the file it was going into baked into
// the separator. Each occupies a line of its own, and becomes a range over the
// hosts with that indentation back where it belongs.
func loopBlock(line string) ([]string, bool) {
	indent := leadingSpace(line)
	switch strings.TrimSpace(line) {
	case "{{{nginx/host_gateways}}}":
		return []string{
			indent + "{{{- range $host := .nginx.hosts}}}",
			indent + `- "{{{$host.name}}}:host-gateway"`,
			indent + "{{{- end}}}",
		}, true
	case "{{{nginx/host_names_with_codes}}}":
		return []string{
			indent + "{{{- range $host := .nginx.hosts}}}",
			indent + "{{{$host.name}}} {{{$host.code}}};",
			indent + "{{{- end}}}",
		}, true
	}
	return nil, false
}

var indentedIncludeRe = regexp.MustCompile(`^([ \t]+)\{\{\{include\s+([^{}\s]+)\s*\}\}\}[ \t]*$`)

// indentedInclude converts an include that sits alone on an indented line, and
// takes the indentation with it.
//
// Every compose file is a column of these — `  {{{include snippets/…/php.yml}}}`
// — and the two spaces belong to nothing. The snippet carries its own
// indentation; the caller's was only ever the gap the splice happened to leave.
// Keeping it means a switched-off service leaves a line of trailing whitespace
// behind in the generated file, thirty of them in a default project, so the
// trim marker takes the line with it when the snippet renders empty.
//
// An include at column zero is a different thing — a Dockerfile splicing in its
// header — and keeps its newline, because the snippet there starts with content
// rather than with a conditional.
func indentedInclude(line string) (string, bool) {
	match := indentedIncludeRe.FindStringSubmatch(line)
	if match == nil {
		return "", false
	}
	return fmt.Sprintf(`%s{{{- template %q .}}}`, match[1], match[2]), true
}

// condition turns the body of an <<<if>>> tag into an expression.
//
// The old tag had no operators at all: it was a run of placeholders, and the
// engine answered "does the substituted text contain the word false". A run
// therefore meant AND, and that is the only thing to reproduce — except for the
// keys that were never configuration, listed in derivedKeys below.
func condition(body string, notes *[]string) string {
	matches := placeholderRe.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		*notes = append(*notes, fmt.Sprintf("condition %q has no placeholder in it and was left as false", body))
		return "false"
	}

	// Anything outside the placeholders was matched as a substring by the old
	// engine and almost certainly means something was written by hand.
	if leftover := strings.TrimSpace(placeholderRe.ReplaceAllString(body, "")); leftover != "" {
		*notes = append(*notes, fmt.Sprintf("condition %q carries literal text %q, which the old engine searched for the word \"false\"", body, leftover))
	}

	terms := make([]string, 0, len(matches))
	for _, match := range matches {
		terms = append(terms, expr(match[1], notes))
	}

	if len(terms) == 1 {
		return terms[0]
	}
	return "and " + strings.Join(terms, " ")
}

// derivedKeys are the keys that were never configuration.
//
// A template could not compare two values, so Go computed the answer and wrote
// it into the config map as though the user had set it — four booleans that are
// each really one comparison. They are the reason this migration is worth
// doing, and the same change that converts these deletes them from the Go side.
//
// nginx/host_name_default is the fifth of the same family: "the first host",
// which needs an ordered list and so could not be asked for at all.
var derivedKeys = map[string]string{
	"db/type_is_mysql":           `(eq .db.type "mysql")`,
	"db/type_is_postgresql":      `(eq .db.type "postgresql")`,
	"db/type_is_mongodb":         `(eq .db.type "mongodb")`,
	"main_service_is_php":        `(eq .main_service "php")`,
	"db/use_default_auth_plugin": `(not (and (eq (lower .db.repository) "mysql") (versionGte .db.version "8.4")))`,
	"nginx/host_name_default":    `(index .nginx.hosts 0).name`,
}

// listKeys are the placeholders that were a list joined into a string in Go.
// Two of them own their line and are converted by loopBlock; the third is
// inline and expands to a pipeline of its own.
var listKeys = map[string]string{
	"nginx/host_gateways":         "",
	"nginx/host_names_with_codes": "",
	"nginx/host_names":            `{{{range $i, $host := .nginx.hosts}}}{{{if $i}}} {{{end}}}{{{$host.name}}}{{{end}}}`,
}

// expr converts one placeholder key to the expression that answers it.
func expr(key string, notes *[]string) string {
	key = strings.TrimSpace(key)

	if replacement, derived := derivedKeys[key]; derived {
		return replacement
	}

	if service, isPort := strings.CutPrefix(key, "port/"); isPort {
		return fmt.Sprintf("port %q", service)
	}

	if path, isInclude := strings.CutPrefix(key, "include "); isInclude {
		return fmt.Sprintf("template %q .", strings.TrimSpace(path))
	}

	for _, part := range strings.Split(key, "/") {
		if !isIdentifier(part) {
			*notes = append(*notes, fmt.Sprintf("key %q cannot be read as a field chain: %q is not a name", key, part))
			return "false"
		}
	}

	return "." + strings.ReplaceAll(key, "/", ".")
}

// substitute rewrites the placeholders in ordinary text.
func substitute(text string, notes *[]string) string {
	return placeholderRe.ReplaceAllStringFunc(text, func(match string) string {
		key := strings.TrimSpace(match[3 : len(match)-3])

		// A template comment is not a placeholder. It cannot appear in the old
		// syntax, so only a re-run over an already-converted tree reaches this
		// — which is exactly what a rebase does.
		if strings.HasPrefix(key, "/*") {
			return match
		}

		if pipeline, isList := listKeys[key]; isList {
			// The two block lists are handled a line at a time. Reaching one
			// here means it shared its line with something, which nothing in
			// the tree does today and which a person has to look at.
			if pipeline == "" {
				*notes = append(*notes, fmt.Sprintf("%q shares its line with other text and was left alone — it expands to several lines", key))
				return match
			}
			return pipeline
		}

		return "{{{" + expr(key, notes) + "}}}"
	})
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}

func leadingSpace(s string) string {
	return s[:len(s)-len(strings.TrimLeft(s, " \t"))]
}

func top(stack []string) string {
	if len(stack) == 0 {
		return ""
	}
	return stack[len(stack)-1]
}
