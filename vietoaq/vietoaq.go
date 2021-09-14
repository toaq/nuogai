package vietoaq

import (
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
)

var (
	toneMap = "\u0304\u0301\u0308\u0309\u0302\u0300\u0303"
	vietoaqMap = [7][2]rune{
		{'r', 'l'}, {'p', 'b'}, {'x', 'z'}, {'n', 'm'}, {'t', 'd'}, {'k', 'g'}, {'f', 'v'}}
	RegularSyllable = regexp.MustCompile(
	  `([bcdfghjklmnprstz']?|[cs]h)`    + // onset
	  `([aeiuoyı])`                     + // first vowel of nucleus
	  `([` + toneMap + `]?)`            + // tone
	  `([aeiouyı]{0,2})`                + // remaining nucleus vowels
	  `(q?)`                            ) // regular coda
	VietoaqSyllable = regexp.MustCompile(
	  `([bcdfghjklmnprstxz]|[cs]h)`     + // onset
	  `([aeiuoy]{1,3})`                 + // nucleus
	  `([qrlpbxznmtdkgfv]?)`            ) // Vietoaq coda
)

func toTransform(syll []string, padding bool) string {
	onset, vow, tone, vows, coda :=
		syll[1], syll[2], syll[3], syll[4], syll[5]
	if onset == "" || onset == "'" {
		onset = "x"
	}
	if tone != "" {
		var qful int
		if coda == "q" {
			qful = 1
		} else {
			qful = 0
		}
		coda = string(vietoaqMap[strings.Index(toneMap, tone) / 2][qful])
	}
	return onset + strings.ReplaceAll(vow + vows, "ı", "i") + coda
}

func fromTransform(syll []string, padding bool) string {
	onset, vow, tone, vows, coda :=
		syll[1], syll[2][0:1], "", syll[2][1:], syll[3]
	if coda != "" && coda != "q" {
		codaRune := rune(coda[0])
		var ii, jj int
		for i, arr := range vietoaqMap {
			for j, char := range arr {
				if char == codaRune {
					ii, jj = i, j
					break
				}
			}
		}
		if jj == 1 {
			coda = "q"
		} else {
			coda = ""
		}
		tone = string([]rune(toneMap)[ii])
		if onset == "x" {
			onset = ""
		}
	} else if vow == "i" {
		vow = "ı"
	}
	if onset == "x" {
		if padding || tone != "" {
			onset = ""
		} else {
			onset = "'"
		}
	}
	return onset + norm.NFC.String(vow + tone) +
		strings.ReplaceAll(vows, "i", "ı") + coda
}

func To(regular string) string {
	return syllableTransform(regular, RegularSyllable, toTransform)
}

func From(vietoaq string) string {
	return syllableTransform(vietoaq, VietoaqSyllable, fromTransform)
}

func syllableTransform(input string, r *regexp.Regexp,
	transform func([]string, bool)string) string {
	interleaved := Syllables(strings.ToLower(norm.NFD.String(input)), r)
	var sb strings.Builder
	for i, s := range interleaved {
		if i % 2 == 1 {
			sb.WriteString(transform(s, interleaved[i - 1][0] != "" || i == 1))
		} else {
			sb.WriteString(s[0])
		}
	}
	return norm.NFC.String(sb.String())
}

// returns an array of junk and Toaq, interleaved
func Syllables(s string, r *regexp.Regexp) [][]string {
	acc := [][]string{}
	for {
		bounds := r.FindStringSubmatchIndex(s)
		if bounds == nil {
			break
		}
		preemptive := r.FindStringSubmatchIndex(s[bounds[1] - 1:])
		if preemptive != nil && preemptive[0] == 0 &&
			bounds[len(bounds) - 1] - bounds[len(bounds) - 2] > 0 {
			bounds[1]--
			bounds[len(bounds) - 1]--
		}
		acc = append(acc, []string{s[:bounds[0]]})
		ln := len(bounds) / 2
		contentful := make([]string, ln)
		for j := 0; j < ln; j++ {
			contentful[j] = s[bounds[2 * j]:bounds[2 * j + 1]]
		}
		acc = append(acc, contentful)
		s = s[bounds[1]:]
	}
	acc = append(acc, []string{s})
	return acc
}
