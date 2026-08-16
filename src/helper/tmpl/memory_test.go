package tmpl

import "testing"

func TestParseMemory(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"768M", 768 << 20},
		{"768MB", 768 << 20},
		{"768m", 768 << 20},
		{" 1G ", 1 << 30},
		{"1.5G", 1536 << 20},
		{"512K", 512 << 10},
		{"1048576", 1 << 20},
		{"1048576B", 1 << 20},
	}

	for _, c := range cases {
		got, err := parseMemory(c.in)
		if err != nil {
			t.Errorf("parseMemory(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseMemory(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

// An unreadable budget has to say so rather than silently size a database at
// zero, which is a container that will not start for a reason nobody can see.
func TestParseMemory_RefusesNonsense(t *testing.T) {
	for _, in := range []string{"", "   ", "lots", "M", "-1G", "12X"} {
		if _, err := parseMemory(in); err == nil {
			t.Errorf("parseMemory(%q) was accepted", in)
		}
	}
}

// The defaults have to render to the numbers that were in the file before this
// existed, or every project's generated my.cnf changes on upgrade for no
// reason. 768M is chosen so that the thirds land exactly.
func TestMemShare_DefaultsReproduceTheOldMyCnf(t *testing.T) {
	pool, err := memShare("768M", 2, 3, "M")
	if err != nil {
		t.Fatal(err)
	}
	if pool != "512M" {
		t.Errorf("innodb_buffer_pool_size = %s, want 512M", pool)
	}

	logBuffer, err := memShare("768M", 1, 3, "M")
	if err != nil {
		t.Fatal(err)
	}
	if logBuffer != "256M" {
		t.Errorf("innodb_log_buffer_size = %s, want 256M", logBuffer)
	}
}

func TestMemShare_Units(t *testing.T) {
	cases := []struct {
		budget      string
		numerator   int
		denominator int
		unit        string
		want        string
	}{
		{"768M", 1, 4, "MB", "192MB"},
		{"768M", 3, 4, "MB", "576MB"},
		{"2G", 1, 2, "G", "1G"},
		{"2G", 1, 1, "M", "2048M"},
	}

	for _, c := range cases {
		got, err := memShare(c.budget, c.numerator, c.denominator, c.unit)
		if err != nil {
			t.Errorf("memShare(%q, %d, %d, %q): %v", c.budget, c.numerator, c.denominator, c.unit, err)
			continue
		}
		if got != c.want {
			t.Errorf("memShare(%q, %d/%d, %q) = %s, want %s",
				c.budget, c.numerator, c.denominator, c.unit, got, c.want)
		}
	}
}

// A budget so small that a share rounds to nothing must not render as 0M: a
// database told to use no memory refuses to start.
func TestMemShare_NeverZero(t *testing.T) {
	got, err := memShare("1M", 1, 8, "M")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1M" {
		t.Fatalf("memShare of a tiny budget = %s, want 1M", got)
	}
}

// MongoDB takes a bare number of gigabytes and accepts a fraction. Rounding
// 0.375 up to 1 would hand the container nearly three times the budget.
func TestMemShareGB_KeepsTheFraction(t *testing.T) {
	got, err := memShareGB("768M", 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got != "0.375" {
		t.Fatalf("memShareGB(768M, 1/2) = %s, want 0.375", got)
	}
}

// MongoDB refuses to start below a quarter of a gigabyte, so that is the floor
// rather than whatever the arithmetic produced.
func TestMemShareGB_HonoursMongosMinimum(t *testing.T) {
	got, err := memShareGB("64M", 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got != "0.25" {
		t.Fatalf("memShareGB of a tiny budget = %s, want 0.25", got)
	}
}
