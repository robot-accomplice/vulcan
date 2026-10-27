// Command fakeagent stands in for Claude Code in tests of the harness. It writes a JSON stream in
// the shape of `claude -p --output-format stream-json --verbose`, runs the check command of the
// workspace's arm once, and, with FAKEAGENT_BREAK=1, first makes a change that breaks a rule: it
// opens a database in the orders part.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func emit(v any) {
	b, _ := json.Marshal(v)
	fmt.Println(string(b))
}

func main() {
	arm := "G"
	if _, err := os.Stat("system.vulcan"); err == nil {
		arm = "V"
	}
	emit(map[string]any{"type": "system", "subtype": "init"})
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
		emit(map[string]any{"type": "assistant", "message": map[string]any{"content": []any{
			map[string]any{"type": "tool_use", "id": "e1", "name": "Edit", "input": map[string]any{"file_path": file}}},
			"usage": map[string]any{"input_tokens": 100, "output_tokens": 20}}})
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
	emit(map[string]any{"type": "assistant", "message": map[string]any{"content": []any{
		map[string]any{"type": "tool_use", "id": "c1", "name": "Bash", "input": map[string]any{"command": command}}},
		"usage": map[string]any{"input_tokens": 200, "output_tokens": 30}}})
	out, err := cmd.CombinedOutput()
	emit(map[string]any{"type": "user", "message": map[string]any{"content": []any{
		map[string]any{"type": "tool_result", "tool_use_id": "c1", "is_error": err != nil, "content": string(out)}}}})
	emit(map[string]any{"type": "result", "subtype": "success", "is_error": false, "num_turns": 2, "total_cost_usd": 0.01,
		"usage": map[string]any{"input_tokens": 300, "cache_read_input_tokens": 1000, "output_tokens": 50}})
}
