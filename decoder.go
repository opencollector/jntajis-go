package jntajis

import (
	"fmt"
	"unicode/utf8"
)

type JNTAJISDecoder struct {
	Replacement rune
	siso        bool
	shiftOffset int
	upper       int
}

func grow(b []byte, req int) []byte {
	v := cap(b)
	if v >= req {
		return b
	}
	v = v | (v >> 1)
	v = v | (v >> 2)
	v = v | (v >> 4)
	v = v | (v >> 4)
	v = v | (v >> 8)
	v = v | (v >> 16)
	v = v | (v >> 32)
	v += 1
	for v < req {
		v = (v << 1) - (v >> 1)
		if v < cap(b) {
			panic("should never be happen")
		}
	}
	nb := make([]byte, len(b), v+1)
	copy(nb, b)
	return nb
}

func (d *JNTAJISDecoder) appendReplacement(b []byte, o int) ([]byte, error) {
	if d.Replacement == InvalidRune {
		return b, fmt.Errorf("inconvertible character found at offset %d", o)
	} else {
		b = grow(b, len(b)+4)
		n := utf8.EncodeRune(b[len(b):len(b)+4], d.Replacement)
		b = b[:len(b)+n]
		return b, nil
	}
}

func (d *JNTAJISDecoder) Decode(b []byte, in_ []byte) ([]byte, error) {
	var err error
	i := 0
	for i < len(in_) {
		var c0 int
		if d.upper > 0 {
			c0 = d.upper
			d.upper = 0
		} else {
			c0 = int(in_[i])
			i += 1
		}
		if c0 >= 0x21 && c0 <= 0x7e {
			if i >= len(in_) {
				d.upper = c0
				return b, nil
			}
			c1 := int(in_[i])
			i += 1
			if c1 >= 0x21 && c1 <= 0x7e {
				jis := d.shiftOffset + (c0-0x21)*94 + (c1 - 0x21)
				m := &txMappings[jis]
				if m.class == Reserved {
					b, err = d.appendReplacement(b, i-2)
					if err != nil {
						return b, err
					}
				} else {
					if m.rs[1] == InvalidRune {
						b = grow(b, len(b)+4)
						n := utf8.EncodeRune(b[len(b):len(b)+4], m.rs[0])
						b = b[:len(b)+n]
					} else {
						b = grow(b, len(b)+8)
						n := utf8.EncodeRune(b[len(b):len(b)+4], m.rs[0])
						n += utf8.EncodeRune(b[len(b)+n:len(b)+n+4], m.rs[1])
						b = b[:len(b)+n]
					}
				}
			} else {
				return nil, fmt.Errorf("unexpected byte \\x%02x after \\x%02x at offset %d", c1, c0, i-2)
			}
		} else {
			siso := d.siso
			if c0 == 0x0e && siso {
				d.shiftOffset = 0
			} else if c0 == 0x0f && siso {
				d.shiftOffset = 94 * 94
			} else {
				return nil, fmt.Errorf("unexpected byte \\x%02x at offset %d", c0, i)
			}
		}
	}
	return b, nil
}
