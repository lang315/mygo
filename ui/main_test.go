package ui

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	checkDirty = os.Getenv("MYGO_CHECK_DIRTY") != "off"
	os.Exit(m.Run())
}
