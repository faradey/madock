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
