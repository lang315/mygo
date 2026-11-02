package ui

import (
	"flag"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	flag.Parse()
	// The oracle scans the rows it finds clean: not in a benchmark, which
	// measures what it spares.
	checkDirty = os.Getenv("MYGO_CHECK_DIRTY") != "off"
	if f := flag.Lookup("test.bench"); f != nil && f.Value.String() != "" {
		checkDirty = false
	}
	os.Exit(m.Run())
}
