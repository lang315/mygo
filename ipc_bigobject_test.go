//go:build !js

package mygo

import (
	"fmt"
	"strings"
	"testing"
)

// A page, of a foreign origin as well, sends an argument that is an object of a
// great many members. The envelope is read before anything is trusted, so reading
// it takes a bounded amount of work per member (TestStrictDuplicateCheckIsLinear
// counts it in package jsonlite); here the call is read whole and answered once.
func TestACallWithAHugeObjectArgumentIsReadAndAnswered(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`{"t":"call","id":1,"k":"tok","m":"X.Y","a":[{`)
	for i := range 80000 {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `"k%07d":0`, i)
	}
	sb.WriteString(`}]}`)
	msg := sb.String()

	const prefix = `{"t":"call","id":1,"k":"tok","m":"X.Y","a":[`
	c, err := parseCall([]byte(msg))
	if err != nil || c.id != 1 || c.name != "X.Y" || len(c.args) != 1 || len(c.args[0]) != len(msg)-len(prefix)-len(`]}`) {
		t.Fatalf("parseCall: %v, id %d, %q, %d arguments", err, c.id, c.name, len(c.args))
	}
	for _, trusted := range []bool{true, false} {
		rec := runGoldenCall(bareWindow(!trusted), trusted, msg)
		if len(rec.Replies) != 1 || !strings.Contains(rec.Replies[0], `"id":1`) {
			t.Errorf("trusted=%v: replies %.200q", trusted, rec.Replies)
		}
	}
}
