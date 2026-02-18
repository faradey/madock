package configs

import (
	"testing"
)

func TestProcessConditionals_SimpleTrue(t *testing.T) {
	input := "before<<<iftrue>>>content<<<endif>>>after"
	expected := "beforecontentafter"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_SimpleFalse(t *testing.T) {
	input := "before<<<iffalse>>>content<<<endif>>>after"
	expected := "beforeafter"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_ElseTrue(t *testing.T) {
	input := "<<<iftrue>>>yes<<<else>>>no<<<endif>>>"
	expected := "yes"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_ElseFalse(t *testing.T) {
	input := "<<<iffalse>>>yes<<<else>>>no<<<endif>>>"
	expected := "no"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_ElseWithSurroundingContent(t *testing.T) {
	input := "before<<<iftrue>>>A<<<else>>>B<<<endif>>>after"
	expected := "beforeAafter"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}

	input2 := "before<<<iffalse>>>A<<<else>>>B<<<endif>>>after"
	expected2 := "beforeBafter"
	got2 := processConditionals(input2)
	if got2 != expected2 {
		t.Errorf("got %q, want %q", got2, expected2)
	}
}

func TestProcessConditionals_NestedIfInsideElse(t *testing.T) {
	input := "<<<iffalse>>>A<<<else>>>B<<<iftrue>>>C<<<endif>>>D<<<endif>>>"
	expected := "BCD"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_NestedIfInsideTrueBranch(t *testing.T) {
	input := "<<<iftrue>>>A<<<iffalse>>>B<<<else>>>C<<<endif>>>D<<<endif>>>"
	expected := "ACD"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_NestedElseSkippedAtWrongDepth(t *testing.T) {
	// The inner <<<else>>> belongs to the nested if, not the outer if
	input := "<<<iffalse>>>X<<<iftrue>>>Y<<<else>>>Z<<<endif>>>W<<<else>>>OUTER<<<endif>>>"
	expected := "OUTER"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_MultilineElse(t *testing.T) {
	input := `    environment:
<<<iftrue>>>
      USER: admin
      PASS: secret
<<<else>>>
      ANON: true
<<<endif>>>
    volumes:`
	expected := `    environment:

      USER: admin
      PASS: secret

    volumes:`
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestProcessConditionals_MultilineElseFalse(t *testing.T) {
	input := `    environment:
<<<iffalse>>>
      USER: admin
      PASS: secret
<<<else>>>
      ANON: true
<<<endif>>>
    volumes:`
	expected := `    environment:

      ANON: true

    volumes:`
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestFindElseAtDepth_NoElse(t *testing.T) {
	content := "just some content"
	got := findElseAtDepth(content)
	if got != -1 {
		t.Errorf("got %d, want -1", got)
	}
}

func TestFindElseAtDepth_TopLevel(t *testing.T) {
	content := "before<<<else>>>after"
	got := findElseAtDepth(content)
	if got != 6 { // "before" is 6 chars
		t.Errorf("got %d, want 6", got)
	}
}

func TestFindElseAtDepth_NestedElseIgnored(t *testing.T) {
	// The else inside the nested if should be ignored
	content := "<<<iftrue>>>inner<<<else>>>innerelse<<<endif>>><<<else>>>outerelse"
	got := findElseAtDepth(content)
	expected := len("<<<iftrue>>>inner<<<else>>>innerelse<<<endif>>>")
	if got != expected {
		t.Errorf("got %d, want %d", got, expected)
	}
}

func TestFindElseAtDepth_Empty(t *testing.T) {
	got := findElseAtDepth("")
	if got != -1 {
		t.Errorf("got %d, want -1", got)
	}
}

func TestProcessConditionals_NoElseNoChange(t *testing.T) {
	// Verify original behavior without else is unchanged
	input := "<<<iftrue>>>kept<<<endif>>>"
	expected := "kept"
	got := processConditionals(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}
