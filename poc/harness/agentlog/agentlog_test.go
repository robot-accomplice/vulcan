package agentlog

import (
	"strings"
	"testing"
)

// A stream in the shape of `claude -p --output-format stream-json --verbose`.
const stream = `{"type":"system","subtype":"init","session_id":"s"}
{"type":"assistant","message":{"content":[{"type":"text","text":"I will check."},{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"vulcan check && vulcan test --race=false"}}],"usage":{"input_tokens":10,"cache_read_input_tokens":100,"output_tokens":5}}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","is_error":true,"content":"api/orders/orders.vulcan:4:2: error K0101: leaf Orders imports leaf Billing\nExit code 1"}]}}
{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t2","name":"Edit","input":{"file_path":"x"}}],"usage":{"input_tokens":3,"output_tokens":2}}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t2","content":"ok"}]}}
{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t3","name":"Bash","input":{"command":"vulcan check"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t3","is_error":true,"content":[{"type":"text","text":"build failed: undefined: x"}]}]}}
{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t4","name":"Bash","input":{"command":"vulcan check"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t4","content":""}]}}
{"type":"result","subtype":"success","is_error":false,"num_turns":4,"total_cost_usd":0.25,"usage":{"input_tokens":40,"cache_creation_input_tokens":200,"cache_read_input_tokens":1000,"output_tokens":90},"permission_denials":[{"tool_name":"Bash"}]}
`

func TestParse(t *testing.T) {
	s, err := Parse("V", strings.NewReader(stream))
	if err != nil {
		t.Fatal(err)
	}
	if s.Turns != 4 || s.ToolCalls != 4 || !s.Done || s.CostUSD != 0.25 {
		t.Fatalf("summary %+v", s)
	}
	if s.Provider != "" {
		t.Fatalf("provider %q", s.Provider)
	}
	if s.Denied != 1 {
		t.Fatalf("denied %d", s.Denied)
	}
	if s.TokensIn != 1240 || s.TokensOut != 90 {
		t.Fatalf("tokens %d %d", s.TokensIn, s.TokensOut)
	}
	if len(s.Checks) != 3 {
		t.Fatalf("checks %+v", s.Checks)
	}
	if s.Checks[0].OK || s.Checks[0].Codes["K0101"] != 1 || s.Checks[0].GoErr {
		t.Fatalf("check 1 %+v", s.Checks[0])
	}
	if s.Checks[1].OK || !s.Checks[1].GoErr {
		t.Fatalf("check 2 %+v", s.Checks[1])
	}
	if !s.Checks[2].OK {
		t.Fatalf("check 3 %+v", s.Checks[2])
	}
}

func TestIsCheck(t *testing.T) {
	if !IsCheck("G", "go vet ./... && go test ./...") || IsCheck("G", "vulcan check") || !IsCheck("V", "cd x && vulcan test Orders") {
		t.Fatal("IsCheck")
	}
}

func TestProviderError(t *testing.T) {
	line := `{"type":"result","subtype":"success","is_error":true,"num_turns":2,"terminal_reason":"api_error","api_error_status":429,"result":"API Error: Request rejected (429)"}`
	s, err := Parse("V", strings.NewReader(line+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Provider != "api_error 429" {
		t.Fatalf("provider %q", s.Provider)
	}
}

// Claude Code writes one event per content block, each with the message's id and usage.
func TestSplitMessage(t *testing.T) {
	split := `{"type":"assistant","message":{"id":"m1","content":[{"type":"thinking","thinking":"x"}],"usage":{"input_tokens":10,"cache_read_input_tokens":100,"output_tokens":0}}}
{"type":"assistant","message":{"id":"m1","content":[{"type":"text","text":"I will check."}],"usage":{"input_tokens":10,"cache_read_input_tokens":100,"output_tokens":0}}}
{"type":"assistant","message":{"id":"m1","content":[{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"go vet ./..."}}],"usage":{"input_tokens":10,"cache_read_input_tokens":100,"output_tokens":0}}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":""}]}}
{"type":"assistant","message":{"id":"m2","content":[{"type":"text","text":"Done."}],"usage":{"input_tokens":5,"output_tokens":0}}}
`
	s, err := Parse("G", strings.NewReader(split))
	if err != nil {
		t.Fatal(err)
	}
	if s.Turns != 2 || s.ToolCalls != 1 || len(s.Checks) != 1 || s.TokensIn != 115 {
		t.Fatalf("summary %+v", s)
	}
}

// A stream with --include-partial-messages, stopped before its result: output tokens come from
// each message's last message_delta; the first edit and the first pass after it are found; arm
// L's linter is a check, and its VL-codes are rule codes.
func TestPartialAndFirstEdit(t *testing.T) {
	stream := `{"type":"system","subtype":"init","session_id":"sess-1"}
{"type":"stream_event","event":{"type":"message_start","message":{"id":"m1"}}}
{"type":"stream_event","event":{"type":"message_delta","usage":{"output_tokens":7}}}
{"type":"assistant","message":{"id":"m1","content":[{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"go vet ./... && vulcan-lint ./... && go test ./..."}}],"usage":{"input_tokens":10,"cache_read_input_tokens":90,"output_tokens":0}}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":"ok"}]}}
{"type":"stream_event","event":{"type":"message_start","message":{"id":"m2"}}}
{"type":"stream_event","event":{"type":"message_delta","usage":{"output_tokens":20}}}
{"type":"assistant","message":{"id":"m2","content":[{"type":"tool_use","id":"t2","name":"Write","input":{"file_path":"/tmp/run/scratch/x.sh"}}],"usage":{"input_tokens":5,"output_tokens":0}}}
{"type":"assistant","message":{"id":"m2","content":[{"type":"tool_use","id":"t3","name":"Edit","input":{"file_path":"/tmp/run/workspace/api/orders/orders.go"}}],"usage":{"input_tokens":5,"output_tokens":0}}}
{"type":"stream_event","event":{"type":"message_start","message":{"id":"m3"}}}
{"type":"stream_event","event":{"type":"message_delta","usage":{"output_tokens":3}}}
{"type":"assistant","message":{"id":"m3","content":[{"type":"tool_use","id":"t4","name":"Bash","input":{"command":"vulcan-lint ./..."}}],"usage":{"input_tokens":4,"output_tokens":0}}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t4","is_error":true,"content":"api/orders/orders.go:3:2: VL0101 orders imports billing\nundefined: billing.Ledger"}]}}
{"type":"stream_event","event":{"type":"message_start","message":{"id":"m4"}}}
{"type":"assistant","message":{"id":"m4","content":[{"type":"tool_use","id":"t5","name":"Bash","input":{"command":"go test ./..."}}],"usage":{"input_tokens":6,"output_tokens":0}}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t5","content":"ok"}]}}
`
	r := NewReader("L")
	r.Scratch = "/tmp/run/scratch"
	for _, l := range strings.Split(stream, "\n") {
		r.Line([]byte(l))
	}
	s := r.S
	if s.Session != "sess-1" || s.Turns != 4 || s.Done {
		t.Fatalf("summary %+v", s)
	}
	if s.TokensOut != 30 || s.TokensIn != 115 || s.Cached != 90 {
		t.Fatalf("tokens out %d in %d cached %d", s.TokensOut, s.TokensIn, s.Cached)
	}
	if len(s.Checks) != 3 || s.Checks[1].Codes["VL0101"] != 1 || s.Checks[1].Undefined != 1 || s.Checks[1].Turn != 3 {
		t.Fatalf("checks %+v", s.Checks)
	}
	// The first check passed before any edit, so it is not the first pass.
	if s.FirstEdit != 2 || s.FirstPass != 4 || s.TokensInAtFirstPass != 115 {
		t.Fatalf("first edit %d, first pass %d, tokens %d", s.FirstEdit, s.FirstPass, s.TokensInAtFirstPass)
	}
}
