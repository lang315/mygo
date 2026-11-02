package gpu

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestPackShaders(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	var shaders [][]byte
	for _, n := range []int{0, 1, 17, 4096, 70000} {
		b := make([]byte, n)
		for i := range b {
			b[i] = byte(rng.Intn(4)) // compressible, like bytecode
		}
		shaders = append(shaders, b)
	}
	blob := PackShaders(shaders)
	got, err := UnpackShaders(string(blob), len(shaders))
	if err != nil {
		t.Fatal(err)
	}
	for i := range shaders {
		if !bytes.Equal(got[i], shaders[i]) {
			t.Errorf("shader %d differs after packing", i)
		}
	}
	if !bytes.Equal(blob, PackShaders(shaders)) {
		t.Error("packing is not deterministic")
	}
	if _, err := UnpackShaders(string(blob), len(shaders)+1); err == nil {
		t.Error("a wrong count was accepted")
	}
	if _, err := UnpackShaders(string(blob[:len(blob)/2]), len(shaders)); err == nil {
		t.Error("cut-short blob was accepted")
	}
}
