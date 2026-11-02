package raster

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"testing"
)

// rgbaByByte is what Image.RGBA was: a byte at a time.
func rgbaByByte(m *Image) []byte {
	out := make([]byte, 4*m.W*m.H)
	for y := 0; y < m.H; y++ {
		src := m.Pix[y*m.Stride:]
		dst := out[y*4*m.W:]
		for x := 0; x < m.W; x++ {
			s, d := src[4*x:4*x+4], dst[4*x:4*x+4]
			d[0], d[1], d[2], d[3] = s[2], s[1], s[0], s[3]
		}
	}
	return out
}

// RGBA swaps red and blue as it always did, whatever the size and the stride.
func TestRGBAMatchesTheByteSwap(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for _, size := range [][2]int{{0, 0}, {0, 5}, {5, 0}, {1, 1}, {2, 1}, {3, 3}, {7, 5}, {8, 8}, {9, 2}, {33, 17}, {400, 600}} {
		for _, pad := range []int{0, 1, 4, 7} {
			w, h := size[0], size[1]
			m := &Image{W: w, H: h, Stride: 4*w + pad}
			m.Pix = make([]byte, m.Stride*h)
			rng.Read(m.Pix)
			if got, want := m.RGBA(), rgbaByByte(m); !bytes.Equal(got, want) {
				t.Errorf("%dx%d, stride %d: RGBA differs from the byte swap", w, h, m.Stride)
			}
		}
	}
}

func BenchmarkRGBA(b *testing.B) {
	m := NewImage(1200, 800)
	rand.New(rand.NewSource(1)).Read(m.Pix)
	b.SetBytes(int64(len(m.Pix)))
	b.Run("word", func(b *testing.B) {
		for b.Loop() {
			m.RGBA()
		}
	})
	b.Run("byte", func(b *testing.B) {
		for b.Loop() {
			rgbaByByte(m)
		}
	})
}

// swapRB64 is swapRB two pixels at a time, which BenchmarkRGBA compares.
func swapRB64(dst, src []byte) {
	const rb = 0x00ff00ff00ff00ff
	i := 0
	for ; i+8 <= len(src); i += 8 {
		v := binary.LittleEndian.Uint64(src[i:])
		binary.LittleEndian.PutUint64(dst[i:], v&^rb|(v&0x000000ff000000ff)<<16|(v>>16)&0x000000ff000000ff)
	}
	for ; i+4 <= len(src); i += 4 {
		d, s := dst[i:i+4], src[i:i+4]
		d[0], d[1], d[2], d[3] = s[2], s[1], s[0], s[3]
	}
}

func BenchmarkSwapRB(b *testing.B) {
	src := make([]byte, 4*1200*800)
	rand.New(rand.NewSource(1)).Read(src)
	dst := make([]byte, len(src))
	b.SetBytes(int64(len(src)))
	for name, f := range map[string]func(dst, src []byte){
		"32":   swapRB,
		"64":   swapRB64,
		"byte": func(dst, src []byte) { rgbaRowByByte(dst, src) },
	} {
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				f(dst, src)
			}
		})
	}
}

func rgbaRowByByte(dst, src []byte) {
	for i := 0; i+4 <= len(src); i += 4 {
		d, s := dst[i:i+4], src[i:i+4]
		d[0], d[1], d[2], d[3] = s[2], s[1], s[0], s[3]
	}
}
