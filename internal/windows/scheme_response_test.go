//go:build windows && (amd64 || arm64)

package windows

import (
	"bytes"
	"sync"
	"testing"
)

// The body's memory takes the size the app's handler announced, up to a
// bound, and a body of another length is still whole.
func TestSchemeResponsePresizes(t *testing.T) {
	for _, tc := range []struct {
		name, length, method string
		status               int
		min, max             int
	}{
		{"a megabyte", "1048576", "GET", 200, 1 << 20, 1<<20 + 4096},
		{"no header", "", "GET", 200, 0, 0},
		{"not a number", "lots", "GET", 200, 0, 0},
		{"negative", "-5", "GET", 200, 0, 0},
		{"zero", "0", "GET", 200, 0, 0},
		{"huge", "99999999999", "GET", 200, maxPresized, maxPresized + 4096},
		{"HEAD", "1048576", "HEAD", 200, 0, 0},
		{"204", "1048576", "GET", 204, 0, 0},
		{"304", "1048576", "GET", 304, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &schemeResponse{head: tc.method == "HEAD"}
			defer r.body.free()
			h := map[string][]string{}
			if tc.length != "" {
				h["Content-Length"] = []string{tc.length}
			}
			r.Respond(tc.status, h)
			if c := r.body.size; c < tc.min || c > tc.max {
				t.Errorf("%d bytes reserved, want %d..%d", c, tc.min, tc.max)
			}
		})
	}
}

// A body of any length comes out whole, written the way the protocol package
// writes it (chunks of 256 KB), by one write, or a byte at a time.
func TestResponseBodyIsWhole(t *testing.T) {
	for _, n := range []int{0, 1, 4095, 4096, 4097, 256 << 10, 1<<20 + 1} {
		want := make([]byte, n)
		for i := range want {
			want[i] = byte(i * 7)
		}
		for _, hint := range []int{0, 10, n, 2 * n} {
			for _, chunk := range []int{1 << 30, 256 << 10, 1000, 1} {
				if chunk == 1 && n > 5000 {
					continue
				}
				var b responseBody
				b.start(hint)
				for p := want; len(p) > 0; {
					k := min(len(p), chunk)
					if err := b.write(p[:k]); err != nil {
						t.Fatalf("n %d hint %d chunk %d: %v", n, hint, chunk, err)
					}
					p = p[k:]
				}
				if got := b.take(); !bytes.Equal(got, want) {
					t.Errorf("n %d hint %d chunk %d: %d bytes, want %d (or content differs)", n, hint, chunk, len(got), n)
				}
				b.free()
			}
		}
	}
}

// Nothing is written before the body starts or after the response ended, by a
// write that came late (a request canceled in the middle).
func TestResponseBodyRefusesLateWrites(t *testing.T) {
	var b responseBody
	if err := b.write([]byte("x")); err == nil {
		t.Error("a write before Respond succeeded")
	}
	b.start(0)
	if err := b.write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	b.take()
	if err := b.write([]byte("y")); err == nil {
		t.Error("a write after the body was taken succeeded")
	}
	b.free()
	if err := b.write([]byte("y")); err == nil {
		t.Error("a write after the body was freed succeeded")
	}
	if got := b.take(); got != nil {
		t.Errorf("take after free = %q", got)
	}
	// Written to on the main thread before it responded, a body opens once.
	var c responseBody
	c.startOnce()
	if err := c.write([]byte("z")); err != nil {
		t.Errorf("a write after startOnce: %v", err)
	}
	c.free()
	c.startOnce()
	if err := c.write([]byte("z")); err == nil {
		t.Error("startOnce opened a body that ended")
	}
}

// The goroutine writing and the main thread ending the response may meet at
// any point (a window closing, a request canceled): the writes either land or
// fail, and no write touches freed memory. (-race needs cgo, which the
// project does not use; this runs the meeting many times, with the heap's own
// checks to catch a write after a free.)
func TestResponseBodyEndsWhileWritten(t *testing.T) {
	chunk := bytes.Repeat([]byte("x"), 64<<10)
	for range 300 {
		var b responseBody
		b.start(0)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for b.write(chunk) == nil {
			}
		}()
		for i := 0; i < 3; i++ {
			_ = b.write(chunk)
		}
		b.free()
		wg.Wait()
	}
}

// The body does not live on Go's heap: a megabyte in chunks of the
// protocol package's costs Go next to nothing (its chunk buffer is reused).
func TestResponseBodyAllocatesNothingOfGo(t *testing.T) {
	chunk := bytes.Repeat([]byte("x"), 256<<10)
	n := testing.AllocsPerRun(5, func() {
		var b responseBody
		b.start(1 << 20)
		for range 4 {
			b.write(chunk)
		}
		b.take()
		b.free()
	})
	if n > 3 {
		t.Errorf("%v allocations a megabyte", n)
	}
}

// A body that found no memory is a 502, not the status and headers of the
// whole response over a short body; and a response that ended does not start
// again, which would leave a block nobody frees.
func TestSchemeResponseOutOfMemory(t *testing.T) {
	r := &schemeResponse{}
	defer r.body.free()
	r.Respond(200, map[string][]string{"Content-Length": {"10"}})
	r.body.mu.Lock()
	err := r.body.grow(1 << 60) // more than there is address space for
	r.body.mu.Unlock()
	if err == nil {
		t.Fatal("grow(1<<60) succeeded")
	}
	if status, _, body := r.outcome(); status != 502 || len(body) == 0 {
		t.Errorf("outcome = %d, %q; want a 502 saying why", status, body)
	}

	ok := &schemeResponse{}
	ok.Respond(200, nil)
	ok.body.write([]byte("whole"))
	if status, _, body := ok.outcome(); status != 200 || string(body) != "whole" {
		t.Errorf("outcome = %d, %q; want the response", status, body)
	}
	ok.body.free()
	ok.Respond(200, map[string][]string{"Content-Length": {"1048576"}}) // after the end
	if ok.body.p != 0 || ok.body.open {
		t.Error("Respond after the response ended opened the body again")
	}
}
