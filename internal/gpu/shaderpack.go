package gpu

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"errors"
	"io"
)

// PackShaders packs compiled shaders into one deflated blob, for the
// bytecode that renderers carry (internal/gpu/d3d11): bytecode of a pixel
// shader is 96 KB and deflates to a sixth.
func PackShaders(shaders [][]byte) []byte {
	var raw bytes.Buffer
	for _, s := range shaders {
		raw.Write(binary.LittleEndian.AppendUint32(nil, uint32(len(s))))
		raw.Write(s)
	}
	var out bytes.Buffer
	w, _ := flate.NewWriter(&out, flate.BestCompression)
	w.Write(raw.Bytes())
	w.Close()
	return out.Bytes()
}

// UnpackShaders returns the n shaders that PackShaders packed into blob.
func UnpackShaders(blob string, n int) ([][]byte, error) {
	raw, err := io.ReadAll(flate.NewReader(bytes.NewReader([]byte(blob))))
	if err != nil {
		return nil, err
	}
	out := make([][]byte, 0, n)
	for len(raw) > 0 {
		if len(raw) < 4 {
			return nil, errors.New("gpu: the packed shaders are cut short")
		}
		size := int(binary.LittleEndian.Uint32(raw))
		raw = raw[4:]
		if size > len(raw) {
			return nil, errors.New("gpu: the packed shaders are cut short")
		}
		out = append(out, raw[:size:size])
		raw = raw[size:]
	}
	if len(out) != n {
		return nil, errors.New("gpu: the packed shaders are not the expected ones")
	}
	return out, nil
}
