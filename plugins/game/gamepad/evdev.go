package gamepad

import "encoding/binary"

// evdevSize is the size of evdev's struct input_event, on Linux of word
// bytes: its time, two longs, then __u16 type, __u16 code and __s32 value.
func evdevSize(word int) int { return 2*word + 8 }

// evdevEvent reads the input_event at the start of e, little endian as
// every architecture that reads pads.
func evdevEvent(e []byte, word int) (typ, code uint16, value int32) {
	e = e[2*word:]
	return binary.LittleEndian.Uint16(e), binary.LittleEndian.Uint16(e[2:]), int32(binary.LittleEndian.Uint32(e[4:]))
}
