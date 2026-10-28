// Package agentlog reads the JSON stream of a headless Claude Code run (`claude -p
// --output-format stream-json --verbose`), and gives the measurements of plan section 8 that the
// stream holds: turns, tool calls, tokens, cost, and each run of the check command.
package agentlog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/robot-accomplice/vulcan/poc/harness/record"
)

// Summary is what one stream tells.
type Summary struct {
	Turns     int
	ToolCalls int
	TokensIn  int64
	TokensOut int64
	CostUSD   float64
	Checks    []record.Check
	Denied    int    // tool calls that the permission rules denied
	Provider  string // the API error that ended the run, such as "api_error 429", or ""
	Done      bool   // the stream has its final result
	Error     string // the agent's error, if the run ended with one
}

// IsCheck reports whether a shell command runs the check command of an arm (plan section 2).
func IsCheck(arm, command string) bool {
	if arm == record.ArmV {
		return strings.Contains(command, "vulcan check") || strings.Contains(command, "vulcan test")
	}
	return strings.Contains(command, "go vet") || strings.Contains(command, "go test")
}

var codeRE = regexp.MustCompile(`\bK[0-9]{4}\b`)

type event struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	Message struct {
		Content json.RawMessage `json:"content"`
		Usage   *usage          `json:"usage"`
	} `json:"message"`
	IsError  bool              `json:"is_error"`
	NumTurns int               `json:"num_turns"`
	Cost     float64           `json:"total_cost_usd"`
	Usage    *usage            `json:"usage"`
	Result   string            `json:"result"`
	Denials  []json.RawMessage `json:"permission_denials"`
	Terminal string            `json:"terminal_reason"`
	APIError int               `json:"api_error_status"`
}

type usage struct {
	Input       int64 `json:"input_tokens"`
	CacheCreate int64 `json:"cache_creation_input_tokens"`
	CacheRead   int64 `json:"cache_read_input_tokens"`
	Output      int64 `json:"output_tokens"`
}

type block struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
}

// Reader reads a stream one line at a time, so that the runner can stop a run at its budget.
type Reader struct {
	Arm               string
	S                 Summary
	pending           map[string]bool // tool_use ids of check commands
	usageIn, usageOut int64
}

// NewReader returns a reader for a run of an arm.
func NewReader(arm string) *Reader { return &Reader{Arm: arm, pending: map[string]bool{}} }

// Line reads one line of the stream. A line that is not an event is ignored.
func (r *Reader) Line(line []byte) {
	var ev event
	if json.Unmarshal(line, &ev) != nil {
		return
	}
	switch ev.Type {
	case "assistant":
		r.S.Turns++
		if u := ev.Message.Usage; u != nil {
			r.usageIn += u.Input + u.CacheCreate + u.CacheRead
			r.usageOut += u.Output
		}
		for _, b := range blocks(ev.Message.Content) {
			if b.Type != "tool_use" {
				continue
			}
			r.S.ToolCalls++
			if b.Name == "Bash" {
				var in struct {
					Command string `json:"command"`
				}
				_ = json.Unmarshal(b.Input, &in)
				if IsCheck(r.Arm, in.Command) {
					r.pending[b.ID] = true
				}
			}
		}
	case "user":
		for _, b := range blocks(ev.Message.Content) {
			if b.Type != "tool_result" || !r.pending[b.ToolUseID] {
				continue
			}
			delete(r.pending, b.ToolUseID)
			text := contentText(b.Content)
			c := record.Check{OK: !b.IsError}
			for _, code := range codeRE.FindAllString(text, -1) {
				if c.Codes == nil {
					c.Codes = map[string]int{}
				}
				c.Codes[code]++
			}
			if b.IsError && len(c.Codes) == 0 {
				c.GoErr = true // a failure of go vet, go test, or the build, with no Vulcan code
			}
			r.S.Checks = append(r.S.Checks, c)
		}
	case "result":
		r.S.Done = true
		if ev.NumTurns > 0 {
			r.S.Turns = ev.NumTurns
		}
		r.S.CostUSD = ev.Cost
		r.S.Denied = len(ev.Denials)
		if ev.Terminal == "api_error" || ev.APIError != 0 {
			r.S.Provider = fmt.Sprintf("api_error %d", ev.APIError)
		}
		if u := ev.Usage; u != nil {
			r.S.TokensIn = u.Input + u.CacheCreate + u.CacheRead
			r.S.TokensOut = u.Output
		}
		if ev.IsError {
			r.S.Error = ev.Subtype
			if r.S.Error == "" {
				r.S.Error = "error"
			}
		}
	}
	if !r.S.Done {
		// Until the final result, the sums of the messages are the best estimate.
		r.S.TokensIn, r.S.TokensOut = r.usageIn, r.usageOut
	}
}

func blocks(raw json.RawMessage) []block {
	var bs []block
	if json.Unmarshal(raw, &bs) == nil {
		return bs
	}
	return nil
}

func contentText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		var b strings.Builder
		for _, p := range parts {
			b.WriteString(p.Text)
			b.WriteString("\n")
		}
		return b.String()
	}
	return string(raw)
}

// Parse reads a whole stream.
func Parse(arm string, in io.Reader) (Summary, error) {
	r := NewReader(arm)
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for sc.Scan() {
		r.Line(sc.Bytes())
	}
	return r.S, sc.Err()
}
