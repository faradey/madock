package proxytransform

import "testing"

type stubTransformer struct {
	calledWith string
	returnVal  string
}

func (s *stubTransformer) TransformProxyConf(content string) string {
	s.calledWith = content
	return s.returnVal
}

func TestApplyNoTransformerReturnsInput(t *testing.T) {
	transformers = nil
	in := "server { listen 80; }"
	if got := Apply(in); got != in {
		t.Errorf("Apply with no transformer = %q, want %q", got, in)
	}
}

// Two consumers register independently — routing and TLS have nothing to do
// with each other and both need the generated file. While this was a single
// slot the second registration silently disabled the first.
func TestApplyChainsTransformers(t *testing.T) {
	transformers = nil
	defer func() { transformers = nil }()

	first := &stubTransformer{returnVal: "after-first"}
	second := &stubTransformer{returnVal: "after-second"}
	AddProxyConfTransformer(first)
	AddProxyConfTransformer(second)

	out := Apply("original")

	if first.calledWith != "original" {
		t.Errorf("first transformer received %q, want original", first.calledWith)
	}
	if second.calledWith != "after-first" {
		t.Errorf("second transformer received %q, want the first one's output", second.calledWith)
	}
	if out != "after-second" {
		t.Errorf("Apply returned %q, want after-second", out)
	}
}

// A transformer with nothing to say must not truncate the file for the ones
// after it — "no change" is the common case for TLS on a machine with no
// certificates.
func TestApplyEmptyReturnDoesNotBreakTheChain(t *testing.T) {
	transformers = nil
	defer func() { transformers = nil }()

	AddProxyConfTransformer(&stubTransformer{returnVal: ""})
	last := &stubTransformer{returnVal: "final"}
	AddProxyConfTransformer(last)

	if out := Apply("original"); out != "final" {
		t.Errorf("Apply returned %q, want final", out)
	}
	if last.calledWith != "original" {
		t.Errorf("last transformer received %q, want the untouched input", last.calledWith)
	}
}

// Set is still "this and nothing else" — a caller that means to replace the
// chain must not merely append to it.
func TestSetReplacesTheChain(t *testing.T) {
	transformers = nil
	defer func() { transformers = nil }()

	AddProxyConfTransformer(&stubTransformer{returnVal: "appended"})
	SetProxyConfTransformer(&stubTransformer{returnVal: "only"})

	if out := Apply("original"); out != "only" {
		t.Errorf("Apply returned %q, want only", out)
	}
}

func TestApplyTransformerRuns(t *testing.T) {
	transformers = nil
	stub := &stubTransformer{returnVal: "transformed"}
	SetProxyConfTransformer(stub)
	defer SetProxyConfTransformer(nil)

	in := "original content"
	out := Apply(in)
	if stub.calledWith != in {
		t.Errorf("transformer received %q, want %q", stub.calledWith, in)
	}
	if out != "transformed" {
		t.Errorf("Apply returned %q, want transformed", out)
	}
}

func TestApplyEmptyReturnFallsBackToInput(t *testing.T) {
	transformers = nil
	stub := &stubTransformer{returnVal: ""}
	SetProxyConfTransformer(stub)
	defer SetProxyConfTransformer(nil)

	in := "preserve me"
	if got := Apply(in); got != in {
		t.Errorf("empty transform return should preserve input, got %q", got)
	}
}

// recorder is a per-project transformer that remembers what it was told.
type recorder struct {
	seen []string
}

func (r *recorder) TransformProjectProxyConf(projectName, content string) string {
	r.seen = append(r.seen, projectName)
	return content + "\n# " + projectName
}

// The whole point of the per-project chain: the transformer is told whose
// blocks these are. Asking the working directory instead is what put one
// project's service-route suffix on all seven projects of a production
// installation.
func TestApplyProject_NamesTheProject(t *testing.T) {
	ResetTransformers()
	t.Cleanup(ResetTransformers)

	rec := &recorder{}
	AddProjectProxyConfTransformer(rec)

	if out := ApplyProject("shiplab-shopify", "server {}"); out != "server {}\n# shiplab-shopify" {
		t.Errorf("output = %q, want the project's own rewrite", out)
	}
	if out := ApplyProject("ops-console", "server {}"); out != "server {}\n# ops-console" {
		t.Errorf("second project got %q — each block is rewritten for its own project", out)
	}
	if len(rec.seen) != 2 || rec.seen[0] != "shiplab-shopify" || rec.seen[1] != "ops-console" {
		t.Errorf("transformer saw %v, want both project names in order", rec.seen)
	}
}

// An empty name runs nothing. A transformer asked to rewrite somebody's
// configuration without being told whose is the defect this exists to remove,
// so there is no fallback to the current directory here or anywhere else.
func TestApplyProject_EmptyNameRewritesNothing(t *testing.T) {
	ResetTransformers()
	t.Cleanup(ResetTransformers)

	rec := &recorder{}
	AddProjectProxyConfTransformer(rec)

	if out := ApplyProject("", "server {}"); out != "server {}" {
		t.Errorf("output = %q, want the content untouched", out)
	}
	if len(rec.seen) != 0 {
		t.Errorf("transformer was called with %v — an unnamed project must not reach it", rec.seen)
	}
}

// The two chains are independent: registering one must not silence the other.
func TestBothChainsRun(t *testing.T) {
	ResetTransformers()
	t.Cleanup(ResetTransformers)

	rec := &recorder{}
	AddProjectProxyConfTransformer(rec)
	AddProxyConfTransformer(suffixAppender{"# whole"})

	block := ApplyProject("core-shopify", "server {}")
	if got := Apply(block); got != "server {}\n# core-shopify\n# whole" {
		t.Errorf("got %q, want both rewrites applied in order", got)
	}
}

// suffixAppender is a whole-file transformer for the test above.
type suffixAppender struct{ line string }

func (s suffixAppender) TransformProxyConf(content string) string {
	return content + "\n" + s.line
}
