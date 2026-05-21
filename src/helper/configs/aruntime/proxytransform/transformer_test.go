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
	transformer = nil
	in := "server { listen 80; }"
	if got := Apply(in); got != in {
		t.Errorf("Apply with no transformer = %q, want %q", got, in)
	}
}

func TestApplyTransformerRuns(t *testing.T) {
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
	stub := &stubTransformer{returnVal: ""}
	SetProxyConfTransformer(stub)
	defer SetProxyConfTransformer(nil)

	in := "preserve me"
	if got := Apply(in); got != in {
		t.Errorf("empty transform return should preserve input, got %q", got)
	}
}
