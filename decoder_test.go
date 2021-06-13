package jntajis

import (
	"fmt"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestDecodeSingle(t *testing.T) {
	for i, m := range txMappings {
		if m.class == Reserved {
			continue
		}
		func(m shrinkingTransliterationMapping) {
			s := make([]byte, 8)
			if m.rs[1] == InvalidRune {
				n := utf8.EncodeRune(s, m.rs[0])
				s = s[:n]
			} else {
				n := utf8.EncodeRune(s[:], m.rs[0])
				n += utf8.EncodeRune(s[n:], m.rs[1])
				s = s[:n]
			}
			var b []byte = nil
			if m.jis >= 94*94 {
				b = append(b, 0x0f)
			}
			b = append(b, byte(0x21+(m.jis/94)%94))
			b = append(b, byte(0x21+m.jis%94))
			if m.jis >= 94*94 {
				b = append(b, 0x0e)
			}
			t.Run(fmt.Sprintf("%d: %s (%U %U)", i, string(s), m.rs[0], m.rs[1]), func(t *testing.T) {
				dec := &JNTAJISDecoder{Replacement: InvalidRune, siso: true}
				result, err := dec.Decode(nil, b)
				if assert.NoError(t, err) {
					assert.Equal(t, s, result)
				}
			})
		}(m)
	}
}

func TestDecodeIncompleteHalf(t *testing.T) {
	dec := &JNTAJISDecoder{Replacement: InvalidRune}
	b, err := dec.Decode(nil, []byte{0x21})
	assert.Equal(t, []byte(nil), b)
	assert.NoError(t, err)
	b, err = dec.Decode(nil, []byte{0x22})
	assert.Equal(t, []byte{0xe3, 0x80, 0x81}, b)
}
