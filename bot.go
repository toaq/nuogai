package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"git.uakci.pl/toaq/nuogai/vietoaq"
	"github.com/bwmarrin/discordgo"
	"github.com/eaburns/toaq/ast"
	"github.com/eaburns/toaq/logic"
)

const (
	lozenge = '▯'
	myself  = "490175530537058314"
	HELP    = "\u2003**commands:**" +
		"\n`%` — Toadūa lookup (3 results at a time)" +
		"\n\u2003(`%37` — show 37 results at a time)" +
		"\n\u2003(`%!` — show one result, with extra info)" +
		"\n\u2003(`%!37` — show 37 results, with extra info)" +
		"\n\u2003(`% 59` — show 59th page of results)" +
		"\n`%serial` — fagri's serial predicate engine" +
		"\n`%nui` — uakci's serial predicate engine" +
		"\n\u2003(`%serial` and `%nui` do not accept tone marks)" +
		"\n`%hoe` — Hoelāı renderer (font version: v0.341)" +
		"\n\u2003(`%hoe!` — same as above; raw input)" +
		"\n`%miu` — jelca's semantic parser"
	UNKNOWN = "unknown command — see `%help` for help"
)

var (
	header     = regexp.MustCompile(`^\*\*.*?\*\*: `)
	whitespace = regexp.MustCompile(`[ ]+`)
	toaPort    string
	spePort    string
	nuiPort    string
)

func mustGetenv(name string) (env string) {
	env, ok := os.LookupEnv(name)
	if !ok {
		panic(fmt.Errorf("environment variable %s missing", name))
	}
	return
}

func init() {
	spePort = mustGetenv("SPE_PORT")
	nuiPort = mustGetenv("NUI_PORT")
	toaPort = mustGetenv("TOA_PORT")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func get(uri string) ([]byte, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return []byte{}, err
	}
	cont, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}
	return cont, nil
}

func post(uri string, ct string, body io.Reader) ([]byte, error) {
	resp, err := http.Post(uri, ct, body)
	if err != nil {
		return []byte{}, err
	}
	cont, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}
	return cont, nil
}

func Respond(dg *discordgo.Session, ms *discordgo.MessageCreate) {
	log.Printf("\n* %s", strings.Join(strings.Split(ms.Message.Content, "\n"), "\n  "))
	respond(ms.Message.Content,
		func(i interface{}) {
			switch t := i.(type) {
			case string:
				dg.ChannelMessageSend(ms.Message.ChannelID, t)
			case []byte:
				dg.ChannelMessageSendComplex(ms.Message.ChannelID,
					&discordgo.MessageSend{
						"", nil, false, []*discordgo.File{
							&discordgo.File{
								"toaq.png",
								"image/png",
								bytes.NewReader(t),
							},
						}, nil, nil, nil,
					})
			}
		})
}

func respond(message string, callback func(interface{})) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("%v", r)
			callback("lủı sa hủı tủoı")
		}
	}()
	message = strings.Trim(
		header.ReplaceAllLiteralString(message, ""),
		" \n")
	parts := whitespace.Split(message, -1)
	cmd, args, rest := parts[0], parts[1:], strings.Join(parts[1:], " ")
	if strings.HasPrefix(cmd, "?%") {
		callback(cmd[1:] + " " + vietoaq.From(rest))
		return
	}
	if strings.HasPrefix(cmd, "%") {
		cmd_ := cmd[1:]
		showNotes := false
		if strings.HasPrefix(cmd_, "!") {
			cmd_ = cmd_[1:]
			showNotes = true
		}
		var (
			n   int
			err error
		)
		if len(cmd_) > 0 {
			n, err = strconv.Atoi(cmd_)
		} else {
			if showNotes {
				n = 1
			} else {
				n = 3
			}
		}
		if err == nil {
			if n >= 0 {
				Toadua(args, callback, n, showNotes)
			} else {
				callback("less than zero, that's quite many")
			}
			return
		}
	}
	switch cmd {
	case "?":
		if strings.HasPrefix(strings.Trim(rest, " \n"), "?") {
			return
		}
		callback(vietoaq.From(rest))
	case "%serial":
		if len(rest) == 0 {
			callback("please supply input")
			return
		}
		resp, err := get(fmt.Sprintf("http://localhost:%s/query?%s", spePort, rest))
		if err != nil {
			log.Print(err)
			callback("connectivity error")
			return
		}
		callback(string(resp))
	case "%nui":
		if len(rest) == 0 {
			callback("please supply input")
			return
		}
		u, err := url.Parse(fmt.Sprintf("http://localhost:%s", nuiPort))
		resp, err := post(u.String(), "application/octet-stream",
			bytes.NewBufferString(rest))
		if err != nil {
			log.Print(err)
			callback("connectivity error")
			return
		}
		callback(string(resp))
	case "%help":
		callback(HELP)
	case "%)":
		callback("(%")
	case "%hoe", "%hoe!":
		if len(rest) == 0 {
			callback("please supply input")
			return
		}
		rest = strings.ReplaceAll(rest, "\t", "\\t")
		if cmd == "%hoe" {
			parts := regexp.MustCompile(`[<>]`).Split(rest, -1)
			var sb strings.Builder
			for i := 0; i < len(parts); i++ {
				s := parts[i]
				if i%2 == 1 {
					sb.WriteString("<")
				} else {
					if i != 0 {
						sb.WriteString(">")
					}
					s = Hoekai(s)
				}
				sb.WriteString(s)
			}
			rest = sb.String()
		}
		out, err := exec.Command("convert",
			"-density", "300",
			"-background", "none",
			"-fill", "white",
			"-strokewidth", "2",
			"-stroke", "black",
			"-font", "ToaqScript",
			"-pointsize", "24",
			"pango:"+rest,
			"-bordercolor", "none",
			"-border", "20",
			"png:-").Output()
		if err != nil {
			log.Print(err)
			callback("lủı sa tủoı")
			return
		}
		callback(out)
	case "%miu":
		if len(rest) == 0 {
			callback("please supply input")
			return
		}
		input := strings.TrimSpace(rest)
		p := ast.NewParser(input)
		text, err := p.Text()
		if err != nil {
			callback("syntax error " + err.Error())
			return
		}
		parse := ast.BracesString(text)
		if parse != "" {
			parse += "\n"
		}
		var math string
		stmt := logic.Interpret(text)
		if stmt == nil {
			math = "fragment"
		} else {
			math = logic.PrettyString(stmt)
		}
		callback(parse + math)
	default:
		if strings.HasPrefix(cmd, "%") {
			callback(UNKNOWN)
		}
	}
}

func Toadua(args []string, callback func(interface{}), howMany int, showNotes bool) {
	if len(args) == 0 {
		callback("please supply a query")
		return
	}
	page, err := strconv.Atoi(args[0])
	if err != nil {
		page = 1
	} else {
		args = args[1:]
	}
	query := strings.Join(args, " ")
	mars, err := json.Marshal(struct {
		S string      `json:"action"`
		I interface{} `json:"query"`
	}{
		"search",
		ToaduaQuery(query),
	})
	if err != nil {
		log.Print(err)
		callback("error")
		return
	}
	raw, err := http.Post(fmt.Sprintf(`http://localhost:%s/api`, toaPort),
		"application/json", bytes.NewReader(mars))
	if err != nil {
		log.Print(err)
		callback("connectivity error")
		return
	}
	var resp struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Entries []struct {
			Id    string `json:"id"`
			User  string `json:"user"`
			Head  string `json:"head"`
			Body  string `json:"body"`
			Score int    `json:"score"`
			Notes []struct {
				User    string `json:"user"`
				Content string `json:"content"`
			} `json:"notes"`
		} `json:"results"`
	}
	body, err := ioutil.ReadAll(raw.Body)
	if err != nil {
		log.Print(err)
		callback("connectivity error")
		return
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		log.Print(err)
		callback("parse error")
		return
	}
	if !resp.Success {
		log.Print(resp.Error)
		callback("search failed: " + resp.Error)
		return
	}
	if len(resp.Entries) == 0 {
		callback("results empty")
		return
	}
	first := (page - 1) * howMany
	if len(resp.Entries) <= first {
		callback(fmt.Sprintf("invalid page number (%d results)", len(resp.Entries)))
		return
	}
	last := min(first+howMany, len(resp.Entries))
	var b strings.Builder
	b.Grow(2000)
	fmt.Fprintf(&b, "\u2003(%d–%d/%d)", first+1, last, len(resp.Entries))
	soFar := b.String()
	for _, e := range resp.Entries[first:last] {
		// if i != 0 {
		b.WriteString("\n")
		// }
		// b.WriteString(" — ")
		b.WriteString("**" + e.Head + "**")
		if showNotes {
			fmt.Fprintf(&b, " (%s)", e.User)
		} else if e.User == "official" {
			b.WriteString(" ❦")
		}
		if e.Score != 0 {
			b.WriteString(" ")
			if e.Score > 0 {
				b.WriteString(strings.Repeat("+", e.Score))
			} else {
				b.WriteString(strings.Repeat("−", -e.Score))
			}
		}
		// b.WriteString(" — ")
		b.WriteString("\n\u2003")
		b.WriteString(strings.Join(strings.Split(e.Body, "\n"), "\n\u2003"))
		if showNotes {
			b.WriteString("")
			for _, note := range e.Notes {
				fmt.Fprintf(&b, "\n\u2003\u2003• (%s) %s", note.User, note.Content)
			}
		}
		old := soFar
		soFar = b.String()
		if len(soFar) > 2000 {
			callback(old)
			b.Reset()
			b.WriteString(soFar[len(old):])
		}
	}
	callback(b.String())
}

func ToaduaQuery(s string) interface{} {
	spaced := strings.Split(s, " ")
	andArgs := make([]interface{}, len(spaced))
	for i, andArg := range spaced {
		ored := strings.Split(andArg, "|")
		orArgs := make([]interface{}, len(ored))
		for j, orArg := range ored {
			neg := false
			if strings.HasPrefix(orArg, "!") {
				orArg = orArg[1:]
				neg = true
			}
			parts := strings.SplitN(orArg, ":", 2)
			var term interface{}
			if len(parts) == 1 {
				term = []interface{}{"term", orArg}
			} else {
				if parts[0] == "arity" {
					conv, _ := strconv.Atoi(parts[1])
					term = []interface{}{"arity", conv}
				} else {
					term = []interface{}{parts[0], parts[1]}
				}
			}
			if neg {
				term = []interface{}{"not", term}
			}
			orArgs[j] = term
		}
		if len(orArgs) == 1 {
			andArgs[i] = orArgs[0]
		} else {
			andArgs[i] = append([]interface{}{"or"}, orArgs...)
		}
	}
	if len(andArgs) == 1 {
		return andArgs[0]
	} else {
		return append([]interface{}{"and"}, andArgs...)
	}
}

func Hoekai(s string) string {
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

func main() {
	dg, err := discordgo.New("Bot " + os.Getenv("TOKEN"))
	if err != nil {
		panic(err)
	}
	dg.AddHandler(Respond)
	err = dg.Open()
	if err != nil {
		panic(err)
	}
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	dg.Close()
}
