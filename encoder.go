package jntajis

import "fmt"

const InvalidJISCode = uint32(0xffffffff)

type JNTAJISEncoder struct {
	Replacement uint32
	putJIS      func([]byte, uint32) ([]byte, bool)
}

type JNTAJISIncrementalEncoder struct {
	Replacement uint32
	putJIS      func([]byte, uint32) ([]byte, bool)
	lookahead   []rune
	shiftState  int
	state       int
}

type ConversionMode int

const (
	ConversionModeSISO = ConversionMode(iota)
	ConversionModeMen1
	ConversionModeJISX0208
	ConversionModeTranslit
)

func (c ConversionMode) String() string {
	switch c {
	case ConversionModeSISO:
		return "ConversionModeSISO"
	case ConversionModeMen1:
		return "ConversionModeMen1"
	case ConversionModeJISX0208:
		return "ConversionModeJISX0208"
	case ConversionModeTranslit:
		return "ConversionModeTranslit"
	default:
		return fmt.Sprintf("??? (%d)", c)
	}
}

func lookupRevTable(r rune) (uint32, bool) {
	s, e := 0, len(runeRangeToJISMappings)
	for s < e && e <= len(runeRangeToJISMappings) {
		m := (s + e) / 2
		mm := &runeRangeToJISMappings[m]
		if r < mm.start {
			e = m
			continue
		} else if r > mm.end {
			s = m + 1
			continue
		}
		o := int(r - mm.start)
		if o >= len(mm.jis) {
			return 0, false
		}
		jis := mm.jis[o]
		if jis == InvalidJISCode {
			return 0, false
		}
		return jis, true
	}
	return 0, false
}

func putJISMen1(b []byte, c uint32) ([]byte, bool) {
	men0, ku0, ten0 := c/(94*94), c/94%94, c%94
	if men0 != 0 {
		return b, false
	}
	return append(b, byte(0x21+ku0), byte(0x21+ten0)), true
}

func putJISX0208(b []byte, c uint32) ([]byte, bool) {
	if int(c) >= len(txMappings) {
		return b, false
	}
	switch txMappings[c].class {
	case KanjiLevel1, KanjiLevel2, JISX0208NonKanji:
		ku0, ten0 := c/94%94, c%94
		return append(b, byte(0x21+ku0), byte(0x21+ten0)), true
	default:
		return b, false
	}
}

func putJISX0208Translit(b []byte, c uint32) ([]byte, bool) {
	if int(c) >= len(txMappings) {
		return b, false
	}
	m := &txMappings[c]
	switch m.class {
	case KanjiLevel1, KanjiLevel2, JISX0208NonKanji:
		ku0, ten0 := c/94%94, c%94
		return append(b, byte(0x21+ku0), byte(0x21+ten0)), true
	default:
		if m.txLen > 0 {
			for _, c := range m.txJIS[:m.txLen] {
				ku0, ten0 := c/94%94, c%94
				b = append(b, byte(0x21+ku0), byte(0x21+ten0))
			}
			return b, true
		} else {
			return b, false
		}
	}
}

func putFuncForConversionMode(mode ConversionMode) func([]byte, uint32) ([]byte, bool) {
	switch mode {
	case ConversionModeMen1:
		return putJISMen1
	case ConversionModeJISX0208:
		return putJISX0208
	case ConversionModeTranslit:
		return putJISX0208Translit
	default:
		panic(fmt.Sprintf("unknown mode: %s", mode))
	}
}

func appendReplacement(b []byte, r rune, jis uint32) ([]byte, error) {
	if jis == InvalidJISCode {
		return nil, fmt.Errorf("%c is not convertible to JISX0208", r)
	} else {
		var ok bool
		b, ok = putJISMen1(b, jis)
		if !ok {
			panic("replacement character is neither convertible to JISX0208!")
		}
	}
	return b, nil
}

func (e *JNTAJISIncrementalEncoder) putShift(b []byte, nextShiftState int) []byte {
	if nextShiftState != e.shiftState {
		e.shiftState = nextShiftState
		switch nextShiftState {
		case 0:
			b = append(b, 0x0e)
		case 1:
			b = append(b, 0x0f)
		default:
			panic("should never happen")
		}
	}
	return b
}

func (e *JNTAJISIncrementalEncoder) putJISSISO(b []byte, jis uint32) ([]byte, bool) {
	b = e.putShift(b, int(jis/(94*94)))
	ku0, ten0 := jis/94%94, jis%94
	return append(b, byte(0x21+ku0), byte(0x21+ten0)), true
}

func (e *JNTAJISIncrementalEncoder) Encode(b []byte, m string) ([]byte, error) {
	put := e.putJIS
	for _, r := range m {
		var jis uint32
		var err error
		ok := false
		e.state, jis = smRuneToJISMapping(e.state, r)
		if e.state == -1 {
			b, ok = put(b, jis)
			if !ok {
				b, err = appendReplacement(b, r, e.Replacement)
				if err != nil {
					return b, err
				}
			}
			e.lookahead = e.lookahead[:0]
			e.state = 0
		} else if e.state == 0 {
			e.lookahead = append(e.lookahead, r)
		} else {
			e.lookahead = append(e.lookahead, r)
			continue
		}
		b, err = e.flushLookahead(b)
		if err != nil {
			return b, err
		}
	}
	return b, nil
}

func (e *JNTAJISIncrementalEncoder) flushLookahead(b []byte) ([]byte, error) {
	put := e.putJIS
	for _, r := range e.lookahead {
		var err error
		jis, ok := lookupRevTable(r)
		if ok {
			b, ok = put(b, jis)
		}
		if !ok {
			b, err = appendReplacement(b, r, e.Replacement)
			if err != nil {
				return b, err
			}
		}
	}
	e.state = 0
	e.lookahead = e.lookahead[:0]
	return b, nil
}

func (e *JNTAJISIncrementalEncoder) Flush(b []byte) ([]byte, error) {
	b, err := e.flushLookahead(b)
	if err != nil {
		return b, err
	}
	b = e.putShift(b, 0)
	return b, nil
}

func (e *JNTAJISEncoder) EncodeAsJISX0213Men1(m string) ([]byte, error) {
	put := e.putJIS
	rb := make([]rune, 0, 2)
	b := make([]byte, 0, len(m))
	s := 0
	var jis uint32
	var err error
	for _, r := range m {
		ok := false
		s, jis = smRuneToJISMapping(s, r)
		if s == -1 {
			b, ok = put(b, jis)
			if !ok {
				b, err = appendReplacement(b, r, e.Replacement)
				if err != nil {
					return b, err
				}
			}
			rb = rb[:0]
			s = 0
		} else if s == 0 {
			rb = append(rb, r)
		} else {
			rb = append(rb, r)
			continue
		}
		for _, r := range rb {
			jis, ok = lookupRevTable(r)
			if ok {
				b, ok = put(b, jis)
			}
			if !ok {
				b, err = appendReplacement(b, r, e.Replacement)
				if err != nil {
					return b, err
				}
			}
		}
		rb = rb[:0]
	}
	for _, r := range rb {
		var ok bool
		jis, ok = lookupRevTable(r)
		if ok {
			b, ok = put(b, jis)
		}
		if !ok {
			b, err = appendReplacement(b, r, e.Replacement)
			if err != nil {
				return b, err
			}
		}
	}
	return b, nil
}

func NewJNTAJISEncoder(mode ConversionMode, replacement uint32) *JNTAJISEncoder {
	return &JNTAJISEncoder{
		Replacement: replacement,
		putJIS:      putFuncForConversionMode(mode),
	}
}

func NewJNTAJISIncrementalEncoder(mode ConversionMode, replacement uint32) *JNTAJISIncrementalEncoder {
	e := &JNTAJISIncrementalEncoder{
		Replacement: replacement,
		lookahead:   make([]rune, 0, 2),
		shiftState:  0,
		state:       0,
	}
	if mode == ConversionModeSISO {
		e.putJIS = e.putJISSISO
	} else {
		e.putJIS = putFuncForConversionMode(mode)
	}
	return e
}
