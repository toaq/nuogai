package main

import (
	"fmt"
	"git.uakci.space/toaq/nuogai/vietoaq"
	"strings"
)

func Hoelai(s string) string {
	viet := vietoaq.To(s)
	parts := vietoaq.Syllables(viet, vietoaq.VietoaqSyllable)
	var sb strings.Builder
	for i, part := range parts {
		if i%2 == 0 {
			sb.WriteString(part[0])
			continue
		}
		onset, nucleus, coda := part[1], part[2], part[3]
		switch onset {
		case "ch":
			onset = "w"
		case "sh":
			onset = "x"
		case "x":
			onset = "q"
		}
		diph := ""
		if len(nucleus) >= 2 {
			flag := true
			switch nucleus[len(nucleus)-2:] {
			case "ai":
				diph = "y"
			case "ao":
				diph = "v"
			case "oi":
				diph = "z"
			case "ei":
				diph = "W"
			default:
				flag = false
			}
			if flag {
				nucleus = nucleus[:len(nucleus)-2]
			}
		}
		if len(nucleus) >= 2 {
			diph = strings.ToUpper(nucleus[1:])
			nucleus = nucleus[:1]
		} else if nucleus == "a" && diph == "" {
			nucleus = ""
		}
		fmt.Fprintf(&sb, "%s%s%s%s",
			diph, onset, strings.ToUpper(coda), nucleus)
	}
	return sb.String()
}
