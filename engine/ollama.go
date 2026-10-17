package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

// OllamaDM runs the Dungeon Master on a local model through Ollama's native
// chat API (http://localhost:11434). It uses a shorter prompt and a focused
// subset of the DM tools, which small models handle far more reliably.
type OllamaDM struct{}

const (
	DefaultOllamaURL   = "http://localhost:11434"
	DefaultOllamaModel = "qwen3:14b"
	DefaultOllamaCtx   = 16384
)

func (c *Config) fillOllamaDefaults() {
	if c.Provider == "" {
		c.Provider = "anthropic"
	}
	if c.OllamaURL == "" {
		c.OllamaURL = DefaultOllamaURL
	}
	if c.OllamaModel == "" {
		c.OllamaModel = DefaultOllamaModel
	}
	if c.OllamaCtx == 0 {
		c.OllamaCtx = DefaultOllamaCtx
	}
}

func (e *Engine) modelLabel() string {
	if e.Cfg.Provider == "ollama" {
		return "ollama:" + e.Cfg.OllamaModel
	}
	return e.Cfg.Model
}

// localTools are the tools offered to local models.
var localTools = []string{
	"roll_check", "start_battle", "damage_character", "heal_character", "give_item", "give_yen", "award_xp",
	"set_title", "character_note", "create_npc", "create_quest", "offer_choice", "set_flag", "adjust_reputation", "move_party",
}

var (
	localPromptOnce sync.Once
	localPromptText string
)

func localSystemPrompt() string {
	localPromptOnce.Do(func() {
		var b strings.Builder
		b.WriteString(`You are the Dungeon Master of a multiplayer Jujutsu Kaisen role-playing game. The players are jujutsu sorcerers exploring a shared world and fighting curses. You narrate the world and every non-player character.

Rules:
- Players may try ANYTHING. Say "yes, and": treat every action as a real attempt and make the world react sensibly. Never refuse to answer.
- The game engine owns dice, HP, yen, items, XP and battles. Never invent numbers. Use tools:
  - roll_check when the outcome is uncertain (DC 10 easy, 13 medium, 15 hard).
  - start_battle when a fight breaks out (curse ids listed below), then stop.
  - give_item / give_yen / award_xp for rewards. set_title when someone gets a job or rank.
  - create_npc for new people, create_quest for new missions, offer_choice for big party decisions.
  - set_flag / adjust_reputation / character_note to remember consequences.
- Write 1-2 short paragraphs in second person, by character name. Never decide what player characters say or feel.
- For a "talk" event, answer ONLY as that NPC, in their voice.
- Canon characters: Gojo (teacher, the strongest), Principal Yaga, Shoko (doctor), Ijichi (driver), Nanami, Maki. Villains: Mahito, Jogo, Hanami, Kenjaku, Sukuna.

Curse ids for start_battle (grade 4 is weakest): `)
		var cs []string
		for _, id := range SortedCurseIDs() {
			t := Bestiary[id]
			if t.Rank >= 5 {
				continue
			}
			cs = append(cs, fmt.Sprintf("%s (%s)", id, t.Grade))
		}
		b.WriteString(strings.Join(cs, ", "))
		b.WriteString("\nUseful item ids: ")
		var items []string
		for id, it := range ItemList {
			if it.Kind == "consumable" || it.Kind == "treasure" || (it.Kind == "weapon" && it.Value > 0 && it.Value < 100000) {
				items = append(items, id)
			}
		}
		sort.Strings(items)
		b.WriteString(strings.Join(items, ", "))
		b.WriteString(". You may also give custom items by name.\n")
		localPromptText = b.String()
	})
	return localPromptText
}

type ollamaMsg struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
	ToolName  string           `json:"tool_name,omitempty"`
}

type ollamaToolCall struct {
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

type ollamaResp struct {
	Message ollamaMsg `json:"message"`
	Error   string    `json:"error"`
}

func ollamaTools() []map[string]any {
	var out []map[string]any
	for _, d := range allToolDefs {
		if !slices.Contains(localTools, d.Name) {
			continue
		}
		params := map[string]any{"type": "object", "properties": d.Props}
		if len(d.Required) > 0 {
			params["required"] = d.Required
		}
		out = append(out, map[string]any{"type": "function", "function": map[string]any{
			"name": d.Name, "description": d.Desc, "parameters": params,
		}})
	}
	return out
}

var (
	thinkRe = regexp.MustCompile(`(?s)<think>.*?</think>`)
	tagRe   = regexp.MustCompile(`</?(says|say|action|campaign|party|surroundings|quests|recent_log|journal|battle)\b[^>]*>`)
)

// cleanLocal strips reasoning and any prompt markup a small model echoed back.
func cleanLocal(s string) string {
	s = thinkRe.ReplaceAllString(s, "")
	s = tagRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

var ollamaHTTP = &http.Client{Timeout: 5 * time.Minute}

func (d *OllamaDM) Handle(ctx context.Context, e *Engine, ev DMEvent) error {
	var url, model, turn, speaker string
	var numCtx int
	builder := &ClaudeDM{compact: true}
	e.Do(func() {
		url, model, numCtx = e.Cfg.OllamaURL, e.Cfg.OllamaModel, e.Cfg.OllamaCtx
		turn = builder.buildTurn(e, ev)
		if ev.Kind == "talk" {
			if n := e.npcByID(ev.Summary); n != nil {
				speaker = n.Name
			}
		}
	})
	actor := ""
	if len(ev.Actions) > 0 {
		actor = ev.Actions[0].CharID
	}
	msgs := []ollamaMsg{{Role: "system", Content: localSystemPrompt()}, {Role: "user", Content: turn}}
	tools := ollamaTools()
	think := strings.HasPrefix(model, "qwen3")
	narrated := false
	for i := 0; i < 8; i++ {
		body := map[string]any{"model": model, "messages": msgs, "tools": tools, "stream": false,
			"options": map[string]any{"num_ctx": numCtx, "temperature": 0.8}}
		if think {
			body["think"] = false // faster replies; the DM doesn't need long private reasoning
		}
		resp, err := ollamaChat(ctx, url, body)
		if err != nil && think && strings.Contains(err.Error(), "think") {
			think = false
			delete(body, "think")
			resp, err = ollamaChat(ctx, url, body)
		}
		if err != nil {
			return err
		}
		m := resp.Message
		text := cleanLocal(m.Content)
		if speaker != "" {
			text = strings.TrimSpace(strings.TrimPrefix(text, speaker+":"))
		}
		if text != "" {
			narrated = true
			e.Do(func() {
				if speaker != "" {
					e.Log("say", speaker, text)
				} else {
					e.Log("dm", "DM", text)
				}
			})
		}
		if len(m.ToolCalls) == 0 {
			break
		}
		msgs = append(msgs, ollamaMsg{Role: "assistant", Content: m.Content, ToolCalls: m.ToolCalls})
		for _, tc := range m.ToolCalls {
			args := string(tc.Function.Arguments)
			// Some models return arguments as a JSON string instead of an object.
			var asString string
			if json.Unmarshal(tc.Function.Arguments, &asString) == nil {
				args = asString
			}
			if !slices.Contains(localTools, tc.Function.Name) {
				msgs = append(msgs, ollamaMsg{Role: "tool", ToolName: tc.Function.Name, Content: "Error: unknown tool " + tc.Function.Name})
				continue
			}
			out, isErr := builder.runTool(e, tc.Function.Name, args, actor)
			if isErr {
				out = "Error: " + out
			}
			msgs = append(msgs, ollamaMsg{Role: "tool", ToolName: tc.Function.Name, Content: out})
		}
	}
	if !narrated && ev.Kind != "arrive" {
		e.Do(func() { e.Log("dm", "DM", "The world waits for your next move.") })
	}
	return nil
}

func ollamaChat(ctx context.Context, url string, body map[string]any) (*ollamaResp, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", url+"/api/chat", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := ollamaHTTP.Do(req)
	if err != nil {
		if ctx.Err() != nil || strings.Contains(err.Error(), "deadline") || strings.Contains(err.Error(), "Timeout") {
			return nil, fmt.Errorf("Ollama took too long to answer — try a smaller model or a smaller memory (context) setting")
		}
		return nil, fmt.Errorf("can't reach Ollama at %s — is it running? (%v)", url, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	var out ollamaResp
	_ = json.Unmarshal(raw, &out)
	if res.StatusCode != 200 || out.Error != "" {
		msg := out.Error
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
		}
		if res.StatusCode == 404 && strings.Contains(msg, "not found") {
			return nil, fmt.Errorf("Ollama doesn't have that model yet — run: ollama pull %v", body["model"])
		}
		return nil, fmt.Errorf("Ollama error: %s", msg)
	}
	return &out, nil
}

// OllamaStatus checks the configured Ollama server and lists installed models.
func (e *Engine) OllamaStatus(ctx context.Context) (installed []string, hasModel bool, err error) {
	var url, model string
	e.Do(func() { url, model = e.Cfg.OllamaURL, e.Cfg.OllamaModel })
	req, _ := http.NewRequestWithContext(ctx, "GET", url+"/api/tags", nil)
	res, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("can't reach Ollama at %s — install it from ollama.com and make sure it's running", url)
	}
	defer res.Body.Close()
	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(res.Body).Decode(&tags); err != nil {
		return nil, false, err
	}
	for _, m := range tags.Models {
		installed = append(installed, m.Name)
		if m.Name == model || strings.TrimSuffix(m.Name, ":latest") == model {
			hasModel = true
		}
	}
	return installed, hasModel, nil
}
