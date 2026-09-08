package project

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/faradey/madock/v4/src/helper/logger"
)

// A heap larger than the container's memory limit kills the container on start,
// and almost nothing says so.
//
// Measured on extmag.com on 2026-09-09: the limit was lowered to 768m while the
// node still took a 1 GB heap from the image's own jvm.options. OpenSearch was
// killed during startup, `setup:upgrade` failed in the deploy that followed, and
// the only visible sign was the service missing from `madock status` — no line
// naming memory, in the deploy output or anywhere else.
//
// So the check is here, at render time, where both numbers are known and where
// refusing costs a message instead of a failed deploy.

// searchMemoryUnits are the suffixes these two settings may carry. Java's own
// -Xmx accepts k/m/g, compose accepts b/k/m/g, and people write 1G and 1g
// interchangeably; all of them are read the same way.
var searchMemoryUnits = []struct {
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

// parseSearchMemory reads one size, or says it cannot.
func parseSearchMemory(value string) (int64, bool) {
	text := strings.TrimSpace(strings.ToUpper(value))
	if text == "" {
		return 0, false
	}

	factor := int64(1)
	for _, unit := range searchMemoryUnits {
		if strings.HasSuffix(text, unit.suffix) {
			factor = unit.factor
			text = strings.TrimSpace(strings.TrimSuffix(text, unit.suffix))
			break
		}
	}

	amount, err := strconv.ParseFloat(text, 64)
	if err != nil || amount < 0 {
		return 0, false
	}
	return int64(amount * float64(factor)), true
}

// checkSearchMemory returns the complaint about one engine's settings, or "".
//
// Three deliberate silences. A value that cannot be parsed is not this check's
// business — the template writes it out and docker says what it thinks; a check
// that guessed here would refuse a form docker accepts. An engine that is off
// has no numbers worth judging. And a heap that merely looks small is left
// alone: OpenSearch says so in its own log, plainly, whereas the kernel killing
// the container says nothing at all.
func checkSearchMemory(engine, heap, limit string) string {
	heapBytes, ok := parseSearchMemory(heap)
	if !ok {
		return ""
	}
	limitBytes, ok := parseSearchMemory(limit)
	if !ok {
		return ""
	}
	if heapBytes < limitBytes {
		return ""
	}

	return fmt.Sprintf(
		"search/%s/heap is %s and search/%s/memory_limit is %s: the JVM cannot fit in the container.\n"+
			"  The container is killed during startup, and what that looks like is the service missing\n"+
			"  from `madock status` — no message about memory anywhere. Measured on a production machine,\n"+
			"  where the deploy that followed failed in setup:upgrade instead.\n"+
			"  Give the limit room above the heap: lucene keeps its own memory outside it.",
		engine, heap, engine, limit)
}

// verifySearchMemory stops the render when either engine is configured to die.
func verifySearchMemory(conf map[string]string) {
	for _, engine := range []string{"opensearch", "elasticsearch"} {
		if conf["search/"+engine+"/enabled"] != "true" {
			continue
		}
		if complaint := checkSearchMemory(engine,
			conf["search/"+engine+"/heap"],
			conf["search/"+engine+"/memory_limit"]); complaint != "" {
			logger.Fatalln(complaint)
		}
	}
}
