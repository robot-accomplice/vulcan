// Command fakeagent stands in for Claude Code in tests of the harness. It writes a JSON stream in
// the shape of `claude -p --output-format stream-json --verbose --include-partial-messages`, runs
// the check command of the workspace's arm once, and, with FAKEAGENT_BREAK=1, first makes a change
// that breaks a rule: it opens a database in the orders part.
//
// When its prompt is the reviewer's (runner.ReviewPrompt), it acts as the reviewer: it rejects
// the first FAKEAGENT_REJECTS reviews, counted in the file FAKEAGENT_STATE, and accepts the rest.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func emit(v any) {
	b, _ := json.Marshal(v)
	fmt.Println(string(b))
}

// message emits one message in the shape of partial messages: its start, its final output
// tokens, and the message itself.
func message(id string, content []any, in, out int) {
	id = fmt.Sprintf("%s-%d", id, os.Getpid()) // each invocation's messages are its own
	emit(map[string]any{"type": "stream_event", "event": map[string]any{"type": "message_start", "message": map[string]any{"id": id}}})
	emit(map[string]any{"type": "stream_event", "event": map[string]any{"type": "message_delta", "usage": map[string]any{"output_tokens": out}}})
	emit(map[string]any{"type": "assistant", "message": map[string]any{"id": id, "content": content,
		"usage": map[string]any{"input_tokens": in, "output_tokens": 0}}})
}

func main() {
	prompt := os.Args[len(os.Args)-1]
	if strings.HasPrefix(prompt, "You review a change") {
		review()
		return
	}
	arm := "G"
	if _, err := os.Stat("system.vulcan"); err == nil {
		arm = "V"
	}
	emit(map[string]any{"type": "system", "subtype": "init", "session_id": "fake-session"})
	if os.Getenv("FAKEAGENT_BREAK") == "1" {
		file := "api/orders/orders.go"
		if arm == "V" {
			file = "api/orders/orders.vulcan"
		}
		b, err := os.ReadFile(file)
		if err == nil {
			s := strings.Replace(string(b), "import (\n", "import (\n\t\"database/sql\"\n", 1)
			s += "\n// probe opens a database where it should not.\nfunc probe() { _, _ = sql.Open(\"sqlite\", \"x.db\") }\n\nvar _ = probe\n"
			_ = os.WriteFile(filepath.FromSlash(file), []byte(s), 0o644)
		}
		message("m-edit", []any{map[string]any{"type": "tool_use", "id": "e1", "name": "Edit", "input": map[string]any{"file_path": file}}}, 100, 20)
		emit(map[string]any{"type": "user", "message": map[string]any{"content": []any{
			map[string]any{"type": "tool_result", "tool_use_id": "e1", "content": "edited"}}}})
	}
	command := "go vet ./... && go test ./..."
	var cmd *exec.Cmd
	if arm == "V" {
		command = "vulcan check && vulcan test --race=false"
		cmd = exec.Command("vulcan", "check")
	} else {
		cmd = exec.Command("go", "vet", "./...")
	}
	message("m-check", []any{map[string]any{"type": "tool_use", "id": "c1", "name": "Bash", "input": map[string]any{"command": command}}}, 200, 30)
	out, err := cmd.CombinedOutput()
	emit(map[string]any{"type": "user", "message": map[string]any{"content": []any{
		map[string]any{"type": "tool_result", "tool_use_id": "c1", "is_error": err != nil, "content": string(out)}}}})
	emit(map[string]any{"type": "result", "subtype": "success", "is_error": false, "num_turns": 2, "total_cost_usd": 0.01,
		"result": "done", "usage": map[string]any{"input_tokens": 300, "cache_read_input_tokens": 1000, "output_tokens": 50}})
}

func review() {
	n := 0
	state := os.Getenv("FAKEAGENT_STATE")
	if state != "" {
		b, _ := os.ReadFile(state)
		n, _ = strconv.Atoi(strings.TrimSpace(string(b)))
		_ = os.WriteFile(state, []byte(strconv.Itoa(n+1)), 0o644)
	}
	rejects, _ := strconv.Atoi(os.Getenv("FAKEAGENT_REJECTS"))
	v := `{"verdict": "accept", "broken_rules": [], "comments": "fine"}`
	if n < rejects {
		v = `{"verdict": "reject", "broken_rules": ["R03"], "comments": "orders opens a database"}`
	}
	emit(map[string]any{"type": "system", "subtype": "init", "session_id": "fake-review"})
	message("r1", []any{map[string]any{"type": "text", "text": "Reviewed."}}, 500, 40)
	emit(map[string]any{"type": "result", "subtype": "success", "is_error": false, "num_turns": 1,
		"result": "The change.\n" + v, "usage": map[string]any{"input_tokens": 500, "output_tokens": 40}})
}
