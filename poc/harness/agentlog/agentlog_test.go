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
