package grade

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/robot-accomplice/vulcan/internal/rules"
	"github.com/robot-accomplice/vulcan/poc/harness/record"
)

// With a rules file beside a workspace (docs/poc/plan-r3.md, section 5), a workspace is rejected only
// by a finding that its baseline does not hold, and a seeded case counts for a team rule only if a
// rule that matches it rejected the case. A fresh session decides each match. So this file records
// what fired, exports what that session needs, and reads what it decided.

// Fired is one new finding of a check: a rule of the rules file, or a model check.
type Fired struct {
	Rule string `json:"rule"` // a rule's id, or model-0205 for a model check
	Code string `json:"code"` // the digits of the code, which are the same in both arms
	Part string `json:"part"`
	Text string `json:"text"` // the rule's text, with no word that names an arm
}

// Matches holds a session's decisions: team rule, then fired rule, then match or no match.
type Matches map[string]map[string]bool

var (
	diagV   = regexp.MustCompile(`^(.+?):(\d+):(\d+): error K(\d{4}): (.*)$`)
	diagL   = regexp.MustCompile(`^(.+?):(\d+):(\d+): VL(\d{4}) (.*)$`)
	ruleTag = regexp.MustCompile(`\(rule ([\w.-]+) in rules\.vulcan`)
)

// parseFired reads the new findings from the output of a check. A model check's text loses each word
// that names an arm, because the judge of a match must not see the arm.
func parseFired(out []byte, ws string) []Fired {
	file, _ := rules.ParseProject(mustRead(filepath.Join(ws, rules.FileName)))
	seen := map[Fired]bool{}
	var fired []Fired
	root := strings.ToLower(filepath.ToSlash(ws)) + "/"
	for _, line := range strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n") {
		m := diagV.FindStringSubmatch(line)
		if m == nil {
			m = diagL.FindStringSubmatch(line)
		}
		if m == nil {
			continue
		}
		name := filepath.ToSlash(m[1])
		if strings.HasPrefix(strings.ToLower(name), root) {
			name = name[len(root):]
		}
		f := Fired{Code: m[4], Part: partOfFile(name)}
		if tag := ruleTag.FindStringSubmatch(m[5]); tag != nil {
			f.Rule = tag[1]
			if file != nil {
				f.Text = file.TextOf(tag[1])
			}
		} else {
			f.Rule = "model-" + m[4]
			f.Text = neutral(m[5])
		}
		if !seen[f] {
			seen[f] = true
			fired = append(fired, f)
		}
	}
	slices.SortFunc(fired, func(a, b Fired) int { return strings.Compare(a.Rule+a.Part, b.Rule+b.Part) })
	return fired
}

func mustRead(p string) []byte {
	b, _ := os.ReadFile(p)
	return b
}

// partOfFile gives the part of a file: where an arm declares or makes its wires counts as "wires".
func partOfFile(name string) string {
	if name == "system.vulcan" || strings.HasPrefix(name, "cmd/") {
		return "wires"
	}
	if d := path.Dir(name); d != "." {
		return d
	}
	return name
}

var armWords = regexp.MustCompile(`(?i)[\w./-]*vulcan[\w./-]*|\b(K|VL)\d{4}\b|\bports\.go\b|\bthe manifest\b|\barch\b|\b[\w/-]+\.go\b`)

// neutral removes each word of a message that names an arm.
func neutral(msg string) string {
	return strings.TrimSpace(armWords.ReplaceAllString(msg, "[...]"))
}

// JudgePair is one thing to decide: does the rule that fired state the team rule?
type JudgePair struct {
	TeamRule  string `json:"team_rule"`
	TeamText  string `json:"team_text"`
	FiredRule string `json:"fired_rule"`
	FiredText string `json:"fired_text"`
	Match     *bool  `json:"match"` // the judge writes true or false
}

// JudgeFile is what a fresh session reads and fills in. It names no arm, and it holds no set size.
type JudgeFile struct {
	Instructions string      `json:"instructions"`
	Pairs        []JudgePair `json:"pairs"`
}

const judgeInstructions = "For each pair, decide from the two texts alone: does the rule that fired state what the team rule asks? Write true or false in match. A rule that rejected a case for another reason is no match."

// ExportJudge gives each pair of a team rule and a rule that fired on one of its seeded cases.
func (v *Validation) ExportJudge(meta *record.Meta) *JudgeFile {
	text := map[string]string{}
	for _, r := range meta.Rules {
		text[r.ID] = r.Text
	}
	best := map[[2]string]string{}
	for _, it := range v.Items {
		if it.Kind != "seeded" {
			continue
		}
		team, _, _ := strings.Cut(it.ID, "/")
		for _, f := range it.Fired {
			k := [2]string{team, f.Rule}
			if cur, ok := best[k]; !ok || f.Text < cur {
				best[k] = f.Text
			}
		}
	}
	out := &JudgeFile{Instructions: judgeInstructions}
	for k, t := range best {
		out.Pairs = append(out.Pairs, JudgePair{TeamRule: k[0], TeamText: text[k[0]], FiredRule: k[1], FiredText: t})
	}
	slices.SortFunc(out.Pairs, func(a, b JudgePair) int {
		return strings.Compare(a.TeamRule+"\x00"+a.FiredRule, b.TeamRule+"\x00"+b.FiredRule)
	})
	return out
}

// Save writes the file for the judge.
func (j *JudgeFile) Save(p string) error {
	b, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

// ReadMatches reads a judge's file. A pair with no decision is an error.
func ReadMatches(p string) (Matches, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var j JudgeFile
	if err := json.Unmarshal(b, &j); err != nil {
		return nil, err
	}
	out := Matches{}
	for _, pair := range j.Pairs {
		if pair.Match == nil {
			return nil, fmt.Errorf("%s: no decision for team rule %s and rule %s", p, pair.TeamRule, pair.FiredRule)
		}
		if out[pair.TeamRule] == nil {
			out[pair.TeamRule] = map[string]bool{}
		}
		out[pair.TeamRule][pair.FiredRule] = *pair.Match
	}
	return out, nil
}
