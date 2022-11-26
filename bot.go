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

	"git.uakci.space/toaq/nuogai/vietoaq"
	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/bwmarrin/discordgo"
	"github.com/eaburns/toaq/ast"
	"github.com/eaburns/toaq/logic"
)

const (
	wikiHelpUrl     = "https://toaq.me/Nuogaı"
	wikiPageUrl     = "https://toaq.me/%s"
	wikiCommandsUrl = "https://toaq.me/Discord/Help_text?action=render"
	toaduaUrl       = "%s/api"
	zugaiUrl        = "%s/zugai?to=%s&text=%s"
	lozenge         = '▯'
)

var (
	markdownLinkRe        = regexp.MustCompile(`!?\[(.*)\]\((.*)\)`)
	alphaHyphenRe         = regexp.MustCompile(`^[a-z-]+$`)
	toaduaCmdRe           = regexp.MustCompile(`^%([1-9][0-9]*)?$`)
	toaduaHost, zugaiHost string
)

func mustGetenv(name string) (env string) {
	env, ok := os.LookupEnv(name)
	if !ok {
		panic(fmt.Errorf("environment variable %s missing", name))
	}
	return
}

func init() {
	toaduaHost = mustGetenv("TOADUA_HOST")
	zugaiHost = mustGetenv("ZUGAI_HOST")
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
			content := strings.TrimSpace(r.Text)
			if len(content) == 0 && len(files) == 0 {
				content = "(nothing was returned)"
			} else if len(content) > 2000 {
				reader := strings.NewReader(content)
				i, offset := 0, 0
				for i < 2000 {
					_, size, err := reader.ReadRune()
					if err != nil {
						break
					}
					offset, i = offset+size, i+1
				}
				content = content[:offset]
			}
			dg.ChannelMessageSendComplex(ms.Message.ChannelID, &discordgo.MessageSend{
				Content: content,
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
	if matches := toaduaCmdRe.FindStringSubmatch(cmd); matches != nil {
		n := 1
		if matches[1] != "" {
			var err error
			n, err = strconv.Atoi(matches[1])
			if err != nil {
				returnText("wrong kinda number")
			}
		}
		Toadua(args, n, returnText)
		return
	} else if strings.HasPrefix(cmd, "?") && len(cmd) > 1 && cmd[1] != '?' {
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
		out, err := exec.Command("expand-serial", rest).Output()
		if err != nil {
			log.Print(err)
			returnText("lủı sa tủoı")
			return
		}
		returnText(string(out))
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
	case "%english", "%logic", "%structure", "%tree", "%all":
		sb := &strings.Builder{}
		var file []byte
		var formats []string
		if cmd == "%all" {
			formats = []string{"english", "structure", "logic", "tree"}
		} else {
			formats = []string{cmd[1:]}
		}
		errorCount := 0
		for i, format := range formats {
			res, err := Zugai(restQuery, format)
			if err != nil {
				errorCount++
				if errorCount == len(formats) {
					sb = &strings.Builder{}
				} else if i != len(formats)-1 {
					res = Response{Text: strings.Split(res.Text, "\n")[0]}
				}
			}
			file = res.Image
			if i != 0 {
				sb.WriteRune('\n')
			}
			text := strings.TrimSpace(string(res.Text))
			if len(text) == 0 && len(res.Image) == 0 {
				fmt.Fprintf(sb, "(%%%s returned nothing)", format)
			} else {
				sb.WriteString(text)
			}
		}
		callback(Response{
			Text:  sb.String(),
			Image: file,
		})
	}
}

func Zugai(input, resource string) (Response, error) {
	isTree := resource == "tree"
	internalName := resource
	if isTree {
		internalName = "xbar-png"
	}
	file, err := get(fmt.Sprintf(zugaiUrl, zugaiHost, internalName, input))
	if err != nil {
		err = fmt.Errorf("%s not available: %w", resource, err)
		log.Println(err)
		return Response{Text: err.Error()}, err
	}
	if isTree {
		return Response{Image: file}, nil
	} else {
		return Response{Text: string(file)}, nil
	}
}

type ToaduaRequest struct {
	Action             string      `json:"action"`
	Query              interface{} `json:"query"`
	PreferredScope     string      `json:"preferred_scope"`
	PreferredScopeBias float64     `json:"preferred_scope_bias"`
}

type ToaduaResponse struct {
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

func Toadua(args []string, howMany int, returnText func(string)) {
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
	mars, err := json.Marshal(ToaduaRequest{
		Action:             "search",
		Query:              ToaduaQuery(query),
		PreferredScope:     "en",
		PreferredScopeBias: 16,
	})
	if err != nil {
		log.Print(err)
		returnText("error")
		return
	}
	raw, err := http.Post(fmt.Sprintf(toaduaUrl, toaduaHost),
		"application/json", bytes.NewReader(mars))
	if err != nil {
		log.Print(err)
		returnText(err.Error())
		return
	}
	var resp ToaduaResponse
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
	searchProngs := strconv.Itoa(first + 1)
	if last > first+1 {
		searchProngs += "–" + strconv.Itoa(last)
	}
	fmt.Fprintf(&b, "\u2003(%s/%d)", searchProngs, len(resp.Entries))
	soFar := b.String()
	for _, e := range resp.Entries[first:last] {
		// if i != 0 {
		b.WriteString("\n")
		// }
		// b.WriteString(" — ")
		b.WriteString("**" + e.Head + "**")
		fmt.Fprintf(&b, " (%s)", e.User)
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
		for _, note := range e.Notes {
			fmt.Fprintf(&b, "\n\u2003\u2003• (%s) %s", note.User, note.Content)
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
	dg, err := discordgo.New("Bot " + os.Getenv("NUOGAI_TOKEN"))
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
