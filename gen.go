//go:generate go run gen.go jissyukutaimap1_0_0.xlsx table.go
//go:build ignore
// +build ignore

package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/360EntSecGroup-Skylar/excelize"
)

const shrinkingMapSheetName = "JIS縮退マップ"

const InvalidRune = rune(0x7fffffff)

type JISCharacterClass int

var memoRegexp = regexp.MustCompile("^類似字形([uU]+[0-9a-fA-F]+)は本文字に変換する。")

const (
	Reserved         = JISCharacterClass(0)
	KanjiLevel1      = JISCharacterClass(1)
	KanjiLevel2      = JISCharacterClass(2)
	KanjiLevel3      = JISCharacterClass(3)
	KanjiLevel4      = JISCharacterClass(4)
	JISX0208NonKanji = JISCharacterClass(9)
	JISX0213NonKanji = JISCharacterClass(11)
)

var categoryNameToEnumMap = map[string]JISCharacterClass{
	"非漢字":   JISX0208NonKanji,
	"追加非漢字": JISX0213NonKanji,
	"JIS1水": KanjiLevel1,
	"JIS2水": KanjiLevel2,
	"JIS3水": KanjiLevel3,
	"JIS4水": KanjiLevel4,
}

var enumToClassNameMap = map[JISCharacterClass]string{
	Reserved:         "Reserved",
	KanjiLevel1:      "KanjiLevel1",
	KanjiLevel2:      "KanjiLevel2",
	KanjiLevel3:      "KanjiLevel3",
	KanjiLevel4:      "KanjiLevel4",
	JISX0208NonKanji: "JISX0208NonKanji",
	JISX0213NonKanji: "JISX0213NonKanji",
}

type shrinkingTransliterationMapping struct {
	// packed men-ku-ten code
	JIS int
	// corresponding Unicode character; second element may be filled with InvalidRune
	Rs [2]rune
	// corresponding Unicode character (secondary); second element may be filled with InvalidRune
	SRs [2]rune
	// JIS character class
	Class JISCharacterClass
	// number of characters for the transliterated form
	TxLen byte
	// transliterated form in packed men-ku-ten code
	TxJIS [4]int
	// transliterated form in Unicode
	TxRunes [4]rune
}

type shrinkingTransliterationMappings []*shrinkingTransliterationMapping

func (m shrinkingTransliterationMappings) Len() int {
	return len(m)
}

func (m shrinkingTransliterationMappings) Less(i, j int) bool {
	return m[i].Rs[0] < m[j].Rs[0]
}

func (m shrinkingTransliterationMappings) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

type runeRangeToJISMapping struct {
	Start, End rune
	JIS        []int
}

const codeTemplate = `package {{.package}}

type JISCharacterClass int

const (
	Reserved         = JISCharacterClass(0)
	KanjiLevel1      = JISCharacterClass(1)
	KanjiLevel2      = JISCharacterClass(2)
	KanjiLevel3      = JISCharacterClass(3)
	KanjiLevel4      = JISCharacterClass(4)
	JISX0208NonKanji = JISCharacterClass(9)
	JISX0213NonKanji = JISCharacterClass(11)
)

const InvalidRune = rune(0x7fffffff)

type shrinkingTransliterationMapping struct {
	// packed men-ku-ten code
	jis uint32
	// corresponding Unicode character; second element may be filled with InvalidRune
	rs [2]rune
	// corresponding Unicode character (secondary); second element may be filled with InvalidRune
	srs [2]rune
	// JIS character class 
	class JISCharacterClass
	// number of characters for the transliterated form
	txLen byte
	// transliterated form in packed men-ku-ten code
	txJIS [4]uint32
	// transliterated form in Unicode
	txRunes [4]rune
}

type runeRangeToJISMapping struct {
	start, end rune
	jis []uint32
}

var txMappings = [2 * 94 * 94]shrinkingTransliterationMapping {
	{{- range .txMappings}}
	{
		jis: {{.JIS}},
		rs: [2]rune{{"{"}}{{range $i, $e := .Rs}}{{if $i|lt 0}}, {{end}}{{.}}{{end}}},
		srs: [2]rune{{"{"}}{{range $i, $e := .SRs}}{{if $i|lt 0}}, {{end}}{{.}}{{end}}},
		class: {{.Class | classToName}},
		txLen: {{.TxLen}},
		txJIS: [4]uint32{{"{"}}{{range $i, $e := .TxJIS}}{{if $i|lt 0}}, {{end}}{{.}}{{end}}},
		txRunes: [4]rune{{"{"}}{{range $i, $e := .TxRunes}}{{if $i|lt 0}}, {{end}}{{.}}{{end}}},
	},
	{{- end}}
}

var runeRangeToJISMappings = []runeRangeToJISMapping {
	{{- range .runeRangeToJISMappings}}
	{
		start: {{.Start}},
		end: {{.End}},
		jis: []uint32{
			{{- range .JIS}}
			{{.}},
			{{- end}}
		},
	},
	{{- end}}
}

func smRuneToJISMapping(state int, r rune) (int, uint32) {
	var final uint32
reenter:
	switch state {
	case 0:
		if r < {{(index .runePairsToJisMappings 0).R}} || r > {{(index .runePairsToJisMappings ((len .runePairsToJisMappings)|add -1)).R}} {
			break
		}
		switch r {
		{{- range $i, $m := .runePairsToJisMappings}}
		case {{$m.R}}:
			state = {{$i|add 1}}
		{{- end}}
		}
	{{- range $i, $m := .runePairsToJisMappings}}
	case {{$i|add 1}}:
		switch r {
		{{- range $m.N}}
		case {{index .Rs 1}}:
			final = {{.JIS}}
			state = -1
		{{- end}}
		default:
			state = 0
			goto reenter
		}
	{{- end}}
	}
	return state, final
}
`

func parseMenKuTenRepr(v string) (int, error) {
	var men, ku, ten int
	_, err := fmt.Sscanf(v, "%d-%d-%d", &men, &ku, &ten)
	if err != nil {
		return 0, err
	}
	if men < 1 || men > 2 {
		return 0, fmt.Errorf("invalid men value: %d", men)
	}
	if ku < 1 || ku > 94 {
		return 0, fmt.Errorf("invalid ku value: %d", ku)
	}
	if ten < 1 || ten > 94 {
		return 0, fmt.Errorf("invalid ten value: %d", ten)
	}
	return (men-1)*94*94 + (ku-1)*94 + (ten - 1), nil
}

func parseRuneRepr(v string) (rune, error) {
	var ucp int
	_, err := fmt.Sscanf(v, "u+%x", &ucp)
	if err != nil {
		return InvalidRune, err
	}
	if ucp < 0 || ucp > 0x10ffff {
		return InvalidRune, fmt.Errorf("invalid unicode code point: %08x", ucp)
	}
	return rune(ucp), nil
}

func parseRuneSeqRepr(v string) ([]rune, error) {
	retval := make([]rune, 0, len(v)/6)
	vv := strings.Split(v, " ")
	for _, c := range vv {
		r, err := parseRuneRepr(c)
		if err != nil {
			return nil, err
		}
		retval = append(retval, r)
	}
	return retval, nil
}

func readExcelFile(f string) ([]shrinkingTransliterationMapping, error) {
	var mappings []shrinkingTransliterationMapping

	wb, err := excelize.OpenFile(f)
	if err != nil {
		return nil, err
	}

	rows := wb.GetRows(shrinkingMapSheetName)

	// assert if it is formatted in the expected manner
	if rows[0][0] != "変換元の文字（JISX0213：1-4水）" ||
		rows[0][4] != "コード変換（1対1変換）" ||
		rows[0][7] != "文字列変換（追加非漢字や、1対ｎの文字変換を行う）" ||
		rows[0][16] != "備考" {
		return nil, fmt.Errorf("a column of the first row does not match to the expected values")
	}
	if rows[1][0] != "面区点コード" ||
		rows[1][1] != "Unicode" ||
		rows[1][2] != "字形" ||
		rows[1][3] != "JIS区分" ||
		rows[1][4] != "面区点コード" ||
		rows[1][5] != "Unicode" ||
		rows[1][6] != "字形" ||
		rows[1][7] != "面区点コード①" ||
		rows[1][8] != "面区点コード②" ||
		rows[1][9] != "面区点コード③" ||
		rows[1][10] != "面区点コード④" ||
		rows[1][11] != "Unicode①" ||
		rows[1][12] != "Unicode②" ||
		rows[1][13] != "Unicode③" ||
		rows[1][14] != "Unicode④" ||
		rows[1][15] != "字形" {
		return nil, fmt.Errorf("a column of the second row does not match to the expected values")
	}

	lj := -1
	for ro, row := range rows[2:] {
		if row[0] == "" {
			break
		}
		class, ok := categoryNameToEnumMap[row[3]]
		if !ok {
			return nil, fmt.Errorf("unknown category name: %s", row[3])
		}
		jis, err := parseMenKuTenRepr(row[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse men-ku-ten at row %d: %w", ro+2, err)
		}
		_rs, err := parseRuneSeqRepr(row[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse rune at row %d: %w", ro+2, err)
		}
		if len(_rs) > 2 {
			return nil, fmt.Errorf("failed to parse rune at row %d: %w", ro+2, err)
		}
		rs := [2]rune{InvalidRune, InvalidRune}
		srs := [2]rune{InvalidRune, InvalidRune}
		copy(rs[:], _rs)
		txLen := 0
		var txJIS = [4]int{0, 0, 0, 0}
		var txRunes = [4]rune{0, 0, 0, 0}
		if lj+1 < jis {
			for i := lj + 1; i < jis; i++ {
				mappings = append(mappings, shrinkingTransliterationMapping{
					JIS:   i,
					Rs:    rs,
					SRs:   srs,
					Class: Reserved,
				})
			}
		}
		if row[4] != "" {
			txLen = 1
			if row[5] == "" {
				return nil, fmt.Errorf("non-empty men-ku-ten code followed by empty Unicode at row %d", ro+2)
			}
			txJIS[0], err = parseMenKuTenRepr(row[4])
			if err != nil {
				return nil, fmt.Errorf("failed to parse men-ku-ten at row %d: %w", ro+2, err)
			}
			txRunes[0], err = parseRuneRepr(row[5])
			if err != nil {
				return nil, fmt.Errorf("failed to parse rune at row %d: %w", ro+2, err)
			}
		} else if row[7] != "" {
			if row[11] == "" {
				return nil, fmt.Errorf("empty single-mapping rune followed by empty runes at row %d", ro+2)
			}
			var i int
			for i = 0; i < 4; i++ {
				var err error
				v := row[7+i]
				if v == "" {
					break
				}
				txJIS[i], err = parseMenKuTenRepr(v)
				if err != nil {
					return nil, fmt.Errorf("failed to parse men-ku-ten at row %d: %w", ro+2, err)
				}
			}
			txJISLen := i
			for i = 0; i < 4; i++ {
				var err error
				v := row[11+i]
				if v == "" {
					break
				}
				txRunes[i], err = parseRuneRepr(v)
				if err != nil {
					return nil, fmt.Errorf("failed to parse rune at row %d: %w", ro+2, err)
				}
			}
			if i != txJISLen {
				return nil, fmt.Errorf("number of characters for the transliteration form does not agree between JIS and Unicode at row %d", ro+2)
			}
			txLen = i
		}
		if row[16] != "" {
			m := memoRegexp.FindStringSubmatch(row[16])
			if len(m) > 0 {
				srs[0], err = parseRuneRepr(m[1])
				if err != nil {
					return nil, fmt.Errorf("failed to parse rune in memo (%s) at row %d: %w", row[16], ro+2, err)
				}
			}
		}

		mappings = append(mappings, shrinkingTransliterationMapping{
			JIS:     jis,
			Rs:      rs,
			SRs:     srs,
			Class:   class,
			TxLen:   byte(txLen),
			TxJIS:   txJIS,
			TxRunes: txRunes,
		})
		lj = jis
	}
	return mappings, nil
}

func doIt(dest string, src string, p string) error {
	t := template.New("").Funcs(map[string]interface{}{
		"add": func(x, y int) int {
			return x + y
		},
		"classToName": func(class JISCharacterClass) string {
			return enumToClassNameMap[class]
		},
	})
	t, err := t.Parse(codeTemplate)
	if err != nil {
		return err
	}
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()
	fmt.Fprintf(os.Stderr, "reading %s...\n", src)
	mappings, err := readExcelFile(src)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "building reverse mappings...\n")
	var rm []runeRangeToJISMapping
	x := make(shrinkingTransliterationMappings, 0, len(mappings))
	for i, _ := range mappings {
		m := &mappings[i]
		x = append(x, m)
		if m.SRs[0] != InvalidRune {
			fm := new(shrinkingTransliterationMapping)
			*fm = *m
			fm.Rs = fm.SRs
			x = append(x, fm)
		}
	}
	sort.Sort(x)
	{
		lr := rune(-1)
		sr := rune(-1)
		gapThr := 256
		js := make([]int, 0, 256)
		for _, m := range x {
			if m.Class == Reserved {
				continue
			}
			if m.Rs[1] != InvalidRune {
				continue
			}
			r := m.Rs[0]
			if lr == -1 {
				sr = r
			} else {
				g := int(r - lr)
				if g >= gapThr {
					rm = append(rm, runeRangeToJISMapping{
						Start: sr,
						End:   lr,
						JIS:   js,
					})
					js = make([]int, 0, 256)
					sr = r
				} else {
					for i := 1; i < g; i++ {
						js = append(js, 0xffffffff)
					}
				}
			}
			js = append(js, m.JIS)
			lr = r
		}
		if lr != -1 {
			rm = append(rm, runeRangeToJISMapping{
				Start: sr,
				End:   lr,
				JIS:   js,
			})
		}
	}
	type outer struct {
		R rune
		N []*shrinkingTransliterationMapping
	}
	var rpm []outer
	{
	next:
		for _, m := range x {
			if m.Class == Reserved {
				continue
			}
			if m.Rs[1] == InvalidRune {
				continue
			}
			r := m.Rs[0]
			for i, _ := range rpm {
				if rpm[i].R == r {
					rpm[i].N = append(rpm[i].N, m)
					continue next
				}
			}
			rpm = append(rpm, outer{r, []*shrinkingTransliterationMapping{m}})
		}
	}
	return t.Execute(
		destFile,
		map[string]interface{}{"package": p, "txMappings": mappings, "runeRangeToJISMappings": rm, "runePairsToJisMappings": rpm},
	)
}

func main() {
	flag.Parse()
	src := flag.Arg(0)
	dest := flag.Arg(1)
	if src == "" {
		fmt.Fprintf(os.Stderr, "specify an .xlsx file\n")
		os.Exit(255)
	}
	if dest == "" {
		fmt.Fprintf(os.Stderr, "specify an output file\n")
		os.Exit(255)
	}
	err := doIt(dest, src, "jntajis")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}
