package tmpl

import (
	"fmt"
	"strconv"
	"strings"
)

// This file gives templates arithmetic over a memory budget, so that one
// setting can size a database engine whatever the engine is.
//
// It exists because the alternative was a per-engine number in every template
// and no way to change any of them without copying the file into a project's
// .madock/docker/ — a frozen copy that then drifts from the shipped one in
// silence. The engines also disagree about what they even want: MySQL takes a
// buffer pool and a log buffer, PostgreSQL takes shared buffers and a guess at
// the OS cache, MongoDB takes a single number in gigabytes and, left alone,
// sizes it from the host's RAM rather than from the container's limit.

// memoryUnits are the suffixes a size may carry, in bytes.
var memoryUnits = []struct {
	suffix string
	factor int64
}{
	{"KB", 1 << 10},
	{"MB", 1 << 20},
	{"GB", 1 << 30},
	{"K", 1 << 10},
	{"M", 1 << 20},
	{"G", 1 << 30},
	{"B", 1},
}

// parseMemory reads a size the way a person writes it in configuration:
// 768M, 1.5G, 512MB, or a bare number of bytes. Case does not matter.
func parseMemory(value string) (int64, error) {
	text := strings.TrimSpace(strings.ToUpper(value))
	if text == "" {
		return 0, fmt.Errorf("memory size is empty — set db/memory, for example 768M")
	}

	factor := int64(1)
	for _, unit := range memoryUnits {
		if strings.HasSuffix(text, unit.suffix) {
			factor = unit.factor
			text = strings.TrimSpace(strings.TrimSuffix(text, unit.suffix))
			break
		}
	}

	amount, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not a memory size: expected something like 768M, 1.5G or 512MB", value)
	}
	if amount < 0 {
		return 0, fmt.Errorf("%q is a negative memory size", value)
	}

	return int64(amount * float64(factor)), nil
}

// memShare returns numerator/denominator of a memory budget, rendered in the
// unit asked for.
//
// A fraction rather than a percentage so the arithmetic lands on the numbers a
// person would have written: two thirds of 768M is exactly 512M, where 67% is
// 514M and every generated file would differ from the one before it for no
// reason.
//
// The result is never zero: a database given "0M" refuses to start, which is a
// worse answer to a small budget than the smallest workable one.
func memShare(budget string, numerator, denominator int, unit string) (string, error) {
	bytes, err := parseMemory(budget)
	if err != nil {
		return "", err
	}
	if denominator == 0 {
		return "", fmt.Errorf("memShare: denominator is zero")
	}

	share := bytes * int64(numerator) / int64(denominator)

	factor := int64(1 << 20)
	for _, u := range memoryUnits {
		if strings.EqualFold(u.suffix, unit) {
			factor = u.factor
			break
		}
	}

	amount := share / factor
	if amount < 1 {
		amount = 1
	}

	return strconv.FormatInt(amount, 10) + unit, nil
}

// memShareGB is memShare for MongoDB, which takes its cache size as a bare
// number of gigabytes and accepts a fraction. Rendering it through memShare
// would round a sensible 0.375 down to 1 GB and hand the container three times
// what was asked for.
//
// The floor is MongoDB's own minimum: it refuses to start below 0.25.
func memShareGB(budget string, numerator, denominator int) (string, error) {
	bytes, err := parseMemory(budget)
	if err != nil {
		return "", err
	}
	if denominator == 0 {
		return "", fmt.Errorf("memShareGB: denominator is zero")
	}

	gb := float64(bytes) * float64(numerator) / float64(denominator) / float64(1<<30)
	if gb < 0.25 {
		gb = 0.25
	}

	return strconv.FormatFloat(gb, 'g', -1, 64), nil
}
