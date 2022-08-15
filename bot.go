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
	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/bwmarrin/discordgo"
	"github.com/eaburns/toaq/ast"
	"github.com/eaburns/toaq/logic"
)

const (
	wikiHelpUrl     = "https://toaq.me/Nuogaı"
	wikiPageUrl     = "https://toaq.me/%s"
	wikiCommandsUrl = "https://toaq.me/Discord/Help_text?action=render"
	lozenge         = '▯'
)

var (
	markdownLinkRe = regexp.MustCompile(`!?\[(.*)\]\((.*)\)`)
	alphaHyphenRe  = regexp.MustCompile(`^[a-z-]+$`)
	ports          struct {
		toa, spe, nui string
	}
)

func mustGetenv(name string) (env string) {
	env, ok := os.LookupEnv(name)
	if !ok {
		panic(fmt.Errorf("environment variable %s missing", name))
	}
	return
}

func init() {
	ports.spe = mustGetenv("SPE_PORT")
	ports.nui = mustGetenv("NUI_PORT")
	ports.toa = mustGetenv("TOA_PORT")
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
	if resp.StatusCode != 200 {
		return []byte{}, fmt.Errorf("%s: %s", resp.Status, cont)
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
	if resp.StatusCode != 200 {
		return []byte{}, fmt.Errorf("%s: %s", resp.Status, cont)
	}
	return cont, nil
}

type Response struct {
	Text  string
	Image []byte
}

func Respond(dg *discordgo.Session, ms *discordgo.MessageCreate) {
	if ms.Message.Content == "" {
		return
	}
	own := ms.Author.ID == dg.State.User.ID
	sigil := ">"
	if own {
		sigil = "<"
	}
	log.Printf("\n%s %s", sigil, strings.Join(strings.Split(ms.Message.Content, "\n"), "\n  "))
	if own {
		return
	}
	respond(ms.Message.Content,
		func(r Response) {
			files := make([]*discordgo.File, 0, 1)
			if len(r.Image) > 0 {
				files = append(files, &discordgo.File{
					Name:        "toaq.png",
					ContentType: "image/png",
					Reader:      bytes.NewReader(r.Image),
				})
			}
			dg.ChannelMessageSendComplex(ms.Message.ChannelID, &discordgo.MessageSend{
				Content: r.Text,
				Files:   files,
			})
		})
}

func respond(message string, callback func(Response)) {
	returnText := func(content string) {
		callback(Response{
			Text: content,
		})
	}
	returnFromRequest := func(res []byte, err error) {
		if err != nil {
			log.Println(err)
			returnText(err.Error())
		} else {
			// silly check
			if len(res) >= 8 && bytes.Equal(res[:8], []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}) {
				callback(Response{
					Image: res,
				})
			} else {
				callback(Response{
					Text: string(res),
				})
			}
		}
	}
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("%v", r)
			returnText("lủı sa hủı tủoı")
		}
	}()

	message = strings.TrimSpace(message)
	parts := strings.Fields(message)
	cmd, args, rest := parts[0], parts[1:], strings.Join(parts[1:], " ")
	restQuery := url.QueryEscape(rest)
	if strings.HasPrefix(cmd, "?%") {
		returnText(cmd[1:] + " " + vietoaq.From(rest))
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
				Toadua(args, returnText, n, showNotes)
			} else {
				returnText("less than zero, that's quite many")
			}
			return
		}
	} else if strings.HasPrefix(cmd, "?") && len(cmd) > 1 {
		fragments := strings.Split(strings.TrimSpace(cmd[1:]+" "+rest), "/")
		for i, fragment := range fragments {
			_, offset, err := strings.NewReader(fragment).ReadRune()
			if err != nil {
				continue
			}
			fragments[i] = strings.ToUpper(fragment[:offset]) + strings.ReplaceAll(fragment[offset:], " ", "_")
		}
		returnText(fmt.Sprintf(wikiPageUrl, strings.Join(fragments, "/")))
		return
	} else if strings.HasPrefix(cmd, "!") && len(args) == 0 && alphaHyphenRe.MatchString(cmd[1:]) && cmd != "!iamvoicechat" {
		all, err := get(wikiCommandsUrl)
		if err != nil {
			returnText(err.Error())
			return
		}
		converter := md.NewConverter("", true, nil)
		converted, err := converter.ConvertString(string(all))
		if err != nil {
			returnText(err.Error())
			return
		}
		prefix := fmt.Sprintf("# !%s\n", cmd[1:])
		if cmd[1:] == "all" || cmd[1:] == "commands" {
			names := &strings.Builder{}
			for _, section := range strings.Split(converted, "##")[1:] {
				if strings.HasPrefix(section, "# !") {
					fmt.Fprintf(names, " `%s`", strings.SplitN(section[3:], "\n", 2)[0])
				} else {
					fmt.Fprintf(names, "\n**%s**:", strings.SplitN(section[1:], "\n", 2)[0])
				}
			}
			returnText(strings.TrimSpace(names.String()))
			return
		}
		var selected string
		for _, line := range strings.Split(string(converted), "##") {
			if strings.HasPrefix(line, prefix) {
				selected = line[len(prefix):]
				break
			}
		}
		if selected != "" {
			selected = markdownLinkRe.ReplaceAllString(selected, "$2")
			returnText(selected)
		} else {
			returnText(fmt.Sprintf("could not find !%s", cmd[1:]))
		}

		return
	}

	switch cmd {
	case "%vietoaq":
		returnText(vietoaq.From(rest))
	case "%serial":
		if len(rest) == 0 {
			returnText("please supply input")
			return
		}
		returnFromRequest(get(fmt.Sprintf("http://localhost:%s/query?%s", ports.spe, restQuery)))
	case "%nui":
		if len(rest) == 0 {
			returnText("please supply input")
			return
		}
		u, _ := url.Parse(fmt.Sprintf("http://localhost:%s", ports.nui))
		returnFromRequest(post(u.String(), "application/octet-stream",
			bytes.NewBufferString(rest)))
	case "%help":
		returnText(fmt.Sprintf("<%s>", wikiHelpUrl))
	case "%)":
		returnText("(%")
	case "%hoe", "%hoe!":
		if len(rest) == 0 {
			returnText("please supply input")
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
			returnText("lủı sa tủoı")
			return
		}
		callback(Response{
			Image: out,
		})
	case "%miu":
		if len(rest) == 0 {
			returnText("please supply input")
			return
		}
		input := strings.TrimSpace(rest)
		p := ast.NewParser(input)
		text, err := p.Text()
		if err != nil {
			returnText("syntax error " + err.Error())
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
		returnText(parse + math)
	case "%english", "%logic", "%structure":
		returnFromRequest(get(fmt.Sprintf("https://zugai.toaq.me/zugai?to=%s&text=%s", cmd[1:], restQuery)))
	case "%tree":
		file, err := get(fmt.Sprintf("https://zugai.toaq.me/zugai?to=xbar-png&text=%s", restQuery))
		if err != nil {
			log.Println(err)
			returnText(fmt.Sprintf("diagram not available: %s", err.Error()))
			return
		}
		callback(Response{
			Image: file,
		})
	case "%all":
		sb := &strings.Builder{}
		for i, name := range []string{"english", "structure", "logic"} {
			res, err := get(fmt.Sprintf("https://zugai.toaq.me/zugai?to=%s&text=%s", name, restQuery))
			if err != nil {
				log.Println(err)
				returnText(err.Error())
				return
			}
			if i != 0 {
				sb.WriteRune('\n')
			}
			sb.WriteString(strings.TrimSpace(string(res)))
		}
		file, err := get(fmt.Sprintf("https://zugai.toaq.me/zugai?to=xbar-png&text=%s", restQuery))
		if err != nil {
			log.Println(err)
			fmt.Fprintf(sb, "\ndiagram not available: %v", err.Error())
		}
		callback(Response{
			Text:  sb.String(),
			Image: file,
		})
	}
}

func Toadua(args []string, returnText func(string), howMany int, showNotes bool) {
	if len(args) == 0 {
		returnText("please supply a query")
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
		returnText("error")
		return
	}
	raw, err := http.Post(fmt.Sprintf(`http://localhost:%s/api`, ports.toa),
		"application/json", bytes.NewReader(mars))
	if err != nil {
		log.Print(err)
		returnText(err.Error())
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
		returnText(err.Error())
		return
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		log.Print(err)
		returnText("parse error")
		return
	}
	if !resp.Success {
		log.Print(resp.Error)
		returnText("search failed: " + resp.Error)
		return
	}
	if len(resp.Entries) == 0 {
		returnText("results empty")
		return
	}
	first := (page - 1) * howMany
	if len(resp.Entries) <= first {
		returnText(fmt.Sprintf("invalid page number (%d results)", len(resp.Entries)))
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
			returnText(old)
			b.Reset()
			b.WriteString(soFar[len(old):])
		}
	}
	returnText(b.String())
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
