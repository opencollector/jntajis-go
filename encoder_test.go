package jntajis

import (
	"fmt"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func runePairToString(p [2]rune) string {
	if p[0] != InvalidRune {
		if p[1] == InvalidRune {
			var b [4]byte
			n := utf8.EncodeRune(b[:], p[0])
			return string(b[:n])
		} else {
			var b [8]byte
			n := utf8.EncodeRune(b[:], p[0])
			n += utf8.EncodeRune(b[n:], p[1])
			return string(b[:n])
		}
	} else {
		return ""
	}
}

func TestEncodeJISX0213Men1Single(t *testing.T) {
	for i, m := range txMappings {
		if m.class == Reserved {
			continue
		}
		func() {
			expected, ok := putJISMen1(nil, m.jis)
			if !ok {
				return
			}
			s := runePairToString(m.rs)
			t.Run(fmt.Sprintf("%d: %s (%U %U)", i, s, m.rs[0], m.rs[1]), func(t *testing.T) {
				enc := NewJNTAJISEncoder(ConversionModeMen1, InvalidJISCode)
				result, err := enc.EncodeAsJISX0213Men1(s)
				if assert.NoError(t, err) {
					assert.Equal(t, expected, result)
				}
			})
		}()
	}
}

func TestEncodeJISX0208Single(t *testing.T) {
	for i, m := range txMappings {
		if m.class == Reserved {
			continue
		}
		func(m shrinkingTransliterationMapping) {
			expected, ok := putJISMen1(nil, m.jis)
			if !ok {
				return
			}
			s0 := runePairToString(m.rs)
			s1 := runePairToString(m.srs)
			t.Run(fmt.Sprintf("%d: %s (%U %U)", i, s0, m.rs[0], m.rs[1]), func(t *testing.T) {
				enc := NewJNTAJISEncoder(ConversionModeJISX0208, InvalidJISCode)
				result, err := enc.EncodeAsJISX0213Men1(s0)
				switch m.class {
				case KanjiLevel1, KanjiLevel2, JISX0208NonKanji:
					if assert.NoError(t, err) {
						assert.Equal(t, expected, result)
					}
				default:
					assert.Error(t, err)
				}
			})
			if s1 != "" {
				t.Run(fmt.Sprintf("%d: %s (%U %U)", i, s1, m.srs[0], m.srs[1]), func(t *testing.T) {
					enc := NewJNTAJISEncoder(ConversionModeJISX0208, InvalidJISCode)
					result, err := enc.EncodeAsJISX0213Men1(s1)
					switch m.class {
					case KanjiLevel1, KanjiLevel2, JISX0208NonKanji:
						if assert.NoError(t, err) {
							assert.Equal(t, expected, result)
						}
					default:
						assert.Error(t, err)
					}
				})
			}
		}(m)
	}
}

func TestEncodeJISX0213Unmapped(t *testing.T) {
	cases := []string{
		"\u0000",
		"\u309a",
		"\u298c6",
		"✋",
	}

	for i, case_ := range cases {
		t.Run(fmt.Sprintf("%d: %s", i, case_), func(t *testing.T) {
			enc := NewJNTAJISEncoder(ConversionModeMen1, InvalidJISCode)
			result, err := enc.EncodeAsJISX0213Men1(case_)
			assert.Error(t, err, result)
		})
	}
}

func TestEncodeSeqs(t *testing.T) {
	cases := []struct {
		expected []byte
		err      string
		mode     ConversionMode
		input    string
	}{
		{
			expected: []byte{
				0x21, 0x24,
			},
			err:   "",
			mode:  ConversionModeMen1,
			input: "，",
		},
		{
			expected: []byte{
				0x21, 0x24,
			},
			err:   "",
			mode:  ConversionModeJISX0208,
			input: "，",
		},
		{
			expected: []byte{
				0x21, 0x24,
			},
			err:   "",
			mode:  ConversionModeTranslit,
			input: "，",
		},
		{
			expected: []byte{0x24, 0x74, 0x24, 0x75, 0x24, 0x76},
			err:      "",
			mode:     ConversionModeMen1,
			input:    "ゔゕゖ",
		},
		{
			expected: nil,
			err:      "ゔ is not convertible to JISX0208",
			mode:     ConversionModeJISX0208,
			input:    "ゔゕゖ",
		},
		{
			expected: []byte{0x25, 0x74, 0x25, 0x75, 0x25, 0x76},
			err:      "",
			mode:     ConversionModeTranslit,
			input:    "ゔゕゖ",
		},
		{
			expected: []byte{0x28, 0x41},
			err:      "",
			mode:     ConversionModeMen1,
			input:    "㉑",
		},
		{
			expected: nil,
			err:      "㉑ is not convertible to JISX0208",
			mode:     ConversionModeJISX0208,
			input:    "㉑",
		},
		{
			expected: []byte{0x23, 0x32, 0x23, 0x31},
			err:      "",
			mode:     ConversionModeTranslit,
			input:    "㉑",
		},
		{
			expected: []byte{
				0x7e, 0x7e,
			},
			err:   "",
			mode:  ConversionModeMen1,
			input: "\u7e6b",
		},
		{
			expected: []byte{},
			err:      "\u7e6b is not convertible to JISX0208",
			mode:     ConversionModeJISX0208,
			input:    "\u7e6b",
		},
		{
			expected: []byte{
				0x37, 0x52,
			},
			err:   "",
			mode:  ConversionModeTranslit,
			input: "\u7e6b",
		},
		{
			expected: []byte{
				0x25, 0x38, 0x25, 0x63, 0x25, 0x73, 0x25, 0x2f,
				0x25, 0x6d, 0x21, 0x3c, 0x25, 0x49, 0x25, 0x74,
				0x25, 0x21, 0x25, 0x73, 0x25, 0x40, 0x25, 0x60,
			},
			err:   "",
			mode:  ConversionModeMen1,
			input: "ジャンクロードヴァンダム",
		},
		{
			expected: []byte{
				0x25, 0x38, 0x25, 0x63, 0x25, 0x73, 0x25, 0x2f,
				0x25, 0x6d, 0x21, 0x3c, 0x25, 0x49, 0x25, 0x74,
				0x25, 0x21, 0x25, 0x73, 0x25, 0x40, 0x25, 0x60,
			},
			err:   "",
			mode:  ConversionModeJISX0208,
			input: "ジャンクロードヴァンダム",
		},
		{
			expected: []byte{
				0x25, 0x38, 0x25, 0x63, 0x25, 0x73, 0x25, 0x2f,
				0x25, 0x6d, 0x21, 0x3c, 0x25, 0x49, 0x25, 0x74,
				0x25, 0x21, 0x25, 0x73, 0x25, 0x40, 0x25, 0x60,
			},
			err:   "",
			mode:  ConversionModeTranslit,
			input: "ジャンクロードヴァンダム",
		},
	}

	for i, case_ := range cases {
		t.Run(fmt.Sprintf("%d: %s", i, case_.input), func(t *testing.T) {
			enc := NewJNTAJISEncoder(case_.mode, InvalidJISCode)
			result, err := enc.EncodeAsJISX0213Men1(case_.input)
			if case_.err != "" {
				assert.EqualError(t, err, case_.err)
			} else {
				if assert.NoError(t, err) {
					assert.Equal(t, case_.expected, result)
				}
			}
		})
	}
}

func TestIncrementalEncodeSeqs(t *testing.T) {
	cases := []struct {
		expected        []byte
		expectedAtFlush []byte
		err             string
		mode            ConversionMode
		input           string
	}{
		{
			expected:        []byte{0x21, 0x24},
			expectedAtFlush: []byte{0x21, 0x24},
			err:             "",
			mode:            ConversionModeMen1,
			input:           "，",
		},
		{
			expected:        []byte{0x21, 0x24},
			expectedAtFlush: []byte{0x21, 0x24},
			err:             "",
			mode:            ConversionModeJISX0208,
			input:           "，",
		},
		{
			expected:        []byte{0x21, 0x24},
			expectedAtFlush: []byte{0x21, 0x24},
			err:             "",
			mode:            ConversionModeTranslit,
			input:           "，",
		},
		{
			expected:        []byte{0x24, 0x74, 0x24, 0x75, 0x24, 0x76},
			expectedAtFlush: []byte{0x24, 0x74, 0x24, 0x75, 0x24, 0x76},
			err:             "",
			mode:            ConversionModeMen1,
			input:           "ゔゕゖ",
		},
		{
			expected:        nil,
			expectedAtFlush: nil,
			err:             "ゔ is not convertible to JISX0208",
			mode:            ConversionModeJISX0208,
			input:           "ゔゕゖ",
		},
		{
			expected:        []byte{0x25, 0x74, 0x25, 0x75, 0x25, 0x76},
			expectedAtFlush: []byte{0x25, 0x74, 0x25, 0x75, 0x25, 0x76},
			err:             "",
			mode:            ConversionModeTranslit,
			input:           "ゔゕゖ",
		},
		{
			expected:        []byte{0x28, 0x41},
			expectedAtFlush: []byte{0x28, 0x41},
			err:             "",
			mode:            ConversionModeMen1,
			input:           "㉑",
		},
		{
			expected:        nil,
			expectedAtFlush: nil,
			err:             "㉑ is not convertible to JISX0208",
			mode:            ConversionModeJISX0208,
			input:           "㉑",
		},
		{
			expected:        []byte{0x23, 0x32, 0x23, 0x31},
			expectedAtFlush: []byte{0x23, 0x32, 0x23, 0x31},
			err:             "",
			mode:            ConversionModeTranslit,
			input:           "㉑",
		},
		{
			expected:        []byte{0x7e, 0x7e},
			expectedAtFlush: []byte{0x7e, 0x7e},
			err:             "",
			mode:            ConversionModeMen1,
			input:           "\u7e6b",
		},
		{
			expected:        nil,
			expectedAtFlush: nil,
			err:             "\u7e6b is not convertible to JISX0208",
			mode:            ConversionModeJISX0208,
			input:           "\u7e6b",
		},
		{
			expected:        []byte{0x37, 0x52},
			expectedAtFlush: []byte{0x37, 0x52},
			err:             "",
			mode:            ConversionModeTranslit,
			input:           "\u7e6b",
		},
		{
			expected:        []byte{0x25, 0x38, 0x25, 0x63, 0x25, 0x73},
			expectedAtFlush: []byte{0x25, 0x38, 0x25, 0x63, 0x25, 0x73, 0x25, 0x2f},
			err:             "",
			mode:            ConversionModeMen1,
			input:           "ジャンク",
		},
		{
			expected:        []byte{0x25, 0x38, 0x25, 0x63, 0x25, 0x73},
			expectedAtFlush: []byte{0x25, 0x38, 0x25, 0x63, 0x25, 0x73, 0x25, 0x2f},
			err:             "",
			mode:            ConversionModeJISX0208,
			input:           "ジャンク",
		},
		{
			expected:        []byte{0x25, 0x38, 0x25, 0x63, 0x25, 0x73},
			expectedAtFlush: []byte{0x25, 0x38, 0x25, 0x63, 0x25, 0x73, 0x25, 0x2f},
			err:             "",
			mode:            ConversionModeTranslit,
			input:           "ジャンク",
		},
		{
			expected:        nil,
			expectedAtFlush: nil,
			err:             "\xf0\xa0\x82\x89 is not convertible to JISX0208",
			mode:            ConversionModeMen1,
			input:           "\xf0\xa0\x82\x89",
		},
		{
			expected:        nil,
			expectedAtFlush: nil,
			err:             "\xf0\xa0\x82\x89 is not convertible to JISX0208",
			mode:            ConversionModeJISX0208,
			input:           "\xf0\xa0\x82\x89",
		},
		{
			expected:        nil,
			expectedAtFlush: nil,
			err:             "\xf0\xa0\x82\x89 is not convertible to JISX0208",
			mode:            ConversionModeTranslit,
			input:           "\xf0\xa0\x82\x89",
		},
		{
			expected:        []byte{0x0f, 0x21, 0x21},
			expectedAtFlush: []byte{0x0f, 0x21, 0x21, 0x0e},
			err:             "",
			mode:            ConversionModeSISO,
			input:           "\xf0\xa0\x82\x89",
		},
	}

	for i, case_ := range cases {
		t.Run(fmt.Sprintf("%d: %s", i, case_.input), func(t *testing.T) {
			enc := NewJNTAJISIncrementalEncoder(case_.mode, InvalidJISCode)
			result, err := enc.Encode(nil, case_.input)
			if case_.err != "" {
				assert.EqualError(t, err, case_.err)
			} else {
				if assert.NoError(t, err) {
					assert.Equal(t, case_.expected, result)
					result, err = enc.Flush(result)
					if assert.NoError(t, err) {
						assert.Equal(t, case_.expectedAtFlush, result)
					}
				}
			}
		})
	}
}
