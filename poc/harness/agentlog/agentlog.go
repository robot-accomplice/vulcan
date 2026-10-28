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
	Turns     int // model messages: Claude Code writes one event per content block, so events with the same message id are one turn
	ToolCalls int
	TokensIn  int64
	TokensOut int64
	Cached    int64 // input tokens read from the provider's cache, part of TokensIn
	CostUSD   float64
	Checks    []record.Check
	Denied    int    // tool calls that the permission rules denied
	Provider  string // the API error that ended the run, such as "api_error 429", or ""
	Done      bool   // the stream has its final result
	Error     string // the agent's error, if the run ended with one
	Session   string // the session id, to resume the session (the review loop)
	Result    string // the final answer's text

	// Measures that start at the first edit (round 2, F5): the turn of the first Edit or Write
	// outside the scratch directory, the turn of the first passing check after it, and the input
	// tokens up to that check.
	FirstEdit, FirstPass int
	TokensInAtFirstPass  int64
}

// IsCheck reports whether a shell command runs the check command of an arm (plan section 2;
// plan-r2 section 2).
func IsCheck(arm, command string) bool {
	switch arm {
	case record.ArmV:
		return strings.Contains(command, "vulcan check") || strings.Contains(command, "vulcan test")
	case record.ArmL:
		if strings.Contains(command, "vulcan-lint") {
			return true
		}
	}
	return strings.Contains(command, "go vet") || strings.Contains(command, "go test")
}

// codeRE finds rule codes: Vulcan's K-codes, and the VL-codes of arm L's linter.
var codeRE = regexp.MustCompile(`\b(K|VL)[0-9]{4}\b`)

type event struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
	SessionID string `json:"session_id"`
	Event     *struct {
		Type    string `json:"type"`
		Message *struct {
			ID string `json:"id"`
		} `json:"message"`
		Usage *usage `json:"usage"`
	} `json:"event"` // --include-partial-messages
	Message struct {
		ID      string          `json:"id"`
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
	Arm     string
	Scratch string // the scratch directory: edits there are not edits of the change
	S       Summary
	pending map[string]int  // tool_use ids of check commands -> the turn that ran them
	seen    map[string]bool // message ids already counted

	usageIn, usageOut, cached int64
	// From --include-partial-messages: each message's output tokens, from its last
	// message_delta. A message's own events carry only the output tokens known at its start.
	partial    bool
	curMessage string
	outBy      map[string]int64

	// The totals of earlier rounds: a resumed session (the review loop) reports only its own
	// usage and cost.
	bankIn, bankOut, bankCached int64
	bankCost                    float64
}

// NewRound starts the stream of a resumed session: the totals so far are kept, and the next
// round's usage adds to them.
func (r *Reader) NewRound() {
	r.bankIn, r.bankOut, r.bankCached, r.bankCost = r.S.TokensIn, r.S.TokensOut, r.S.Cached, r.S.CostUSD
	r.usageIn, r.usageOut, r.cached = 0, 0, 0
	r.partial, r.curMessage, r.outBy = false, "", map[string]int64{}
	r.S.Done = false
}

// NewReader returns a reader for a run of an arm.
func NewReader(arm string) *Reader {
	return &Reader{Arm: arm, pending: map[string]int{}, seen: map[string]bool{}, outBy: map[string]int64{}}
}

func (r *Reader) partialOut() int64 {
	var n int64
	for _, v := range r.outBy {
		n += v
	}
	return n
}

// Line reads one line of the stream. A line that is not an event is ignored.
func (r *Reader) Line(line []byte) {
	var ev event
	if json.Unmarshal(line, &ev) != nil {
		return
	}
	switch ev.Type {
	case "system":
		if ev.Subtype == "init" && ev.SessionID != "" {
			r.S.Session = ev.SessionID
		}
	case "stream_event":
		if e := ev.Event; e != nil {
			switch e.Type {
			case "message_start":
				if e.Message != nil {
					r.curMessage = e.Message.ID
				}
			case "message_delta":
				if e.Usage != nil {
					r.partial = true
					r.outBy[r.curMessage] = e.Usage.Output
				}
			}
		}
	case "assistant":
		// The events of one message repeat its usage; count the message once.
		if id := ev.Message.ID; id == "" || !r.seen[id] {
			if id != "" {
				r.seen[id] = true
			}
			r.S.Turns++
			if u := ev.Message.Usage; u != nil {
				r.usageIn += u.Input + u.CacheCreate + u.CacheRead
				r.usageOut += u.Output
				r.cached += u.CacheRead
			}
		}
		for _, b := range blocks(ev.Message.Content) {
			if b.Type != "tool_use" {
				continue
			}
			r.S.ToolCalls++
			if r.S.FirstEdit == 0 && (b.Name == "Edit" || b.Name == "Write" || b.Name == "MultiEdit" || b.Name == "NotebookEdit") {
				var in struct {
					Path string `json:"file_path"`
				}
				_ = json.Unmarshal(b.Input, &in)
				if r.Scratch == "" || !strings.HasPrefix(in.Path, r.Scratch) {
					r.S.FirstEdit = r.S.Turns
				}
			}
			if b.Name == "Bash" {
				var in struct {
					Command string `json:"command"`
				}
				_ = json.Unmarshal(b.Input, &in)
				if IsCheck(r.Arm, in.Command) {
					r.pending[b.ID] = r.S.Turns
				}
			}
		}
	case "user":
		for _, b := range blocks(ev.Message.Content) {
			turn, ok := r.pending[b.ToolUseID]
			if b.Type != "tool_result" || !ok {
				continue
			}
			delete(r.pending, b.ToolUseID)
			text := contentText(b.Content)
			c := record.Check{OK: !b.IsError, Turn: turn, Undefined: strings.Count(text, "undefined:")}
			if c.OK && r.S.FirstPass == 0 && r.S.FirstEdit > 0 && turn >= r.S.FirstEdit {
				r.S.FirstPass = turn
				r.S.TokensInAtFirstPass = r.bankIn + r.usageIn
			}
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
		r.S.Done = true // Turns stays the count of messages: num_turns counts otherwise and is not in a stopped run
		r.S.Result = ev.Result
		r.S.CostUSD = r.bankCost + ev.Cost
		r.S.Denied = len(ev.Denials)
		if ev.Terminal == "api_error" || ev.APIError != 0 {
			r.S.Provider = fmt.Sprintf("api_error %d", ev.APIError)
		}
		if u := ev.Usage; u != nil {
			r.S.TokensIn = r.bankIn + u.Input + u.CacheCreate + u.CacheRead
			r.S.TokensOut = r.bankOut + u.Output
			r.S.Cached = r.bankCached + u.CacheRead
		}
		if ev.IsError {
			r.S.Error = ev.Subtype
			if r.S.Error == "" {
				r.S.Error = "error"
			}
		}
	}
	if !r.S.Done {
		// Until the final result, the sums of the messages are the best estimate; with partial
		// messages, the output tokens are exact.
		r.S.TokensIn, r.S.TokensOut, r.S.Cached = r.bankIn+r.usageIn, r.bankOut+r.usageOut, r.bankCached+r.cached
		if r.partial {
			r.S.TokensOut = r.bankOut + r.partialOut()
		}
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
