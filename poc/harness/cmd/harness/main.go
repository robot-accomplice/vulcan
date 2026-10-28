// Command harness runs the PoC evaluation (docs/poc/plan.md, sections 6 to 9).
//
//	harness schedule -tasks T01,T02 -runs 3 -seed 42 [-arms P,L,V] > schedule.json
//	harness evaluate -schedule schedule.json -material DIR -out DIR [-model M] [-budget-usd X]
//	        [-sandbox DISTRO] [-review] [-prices prices.json]   (round 2, docs/poc/plan-r2.md)
//	harness run -arm V -task T01 -index 1 -brief brief.md -rules RULES.md -out DIR
//	harness grade -out DIR -material DIR
//	harness analyze -runs runs.jsonl -meta meta.json > report.md
//	harness synth -scenario promising -out DIR
//	harness newkey | seal -dir DIR -out FILE | open -in FILE -dir DIR | manifest -dir DIR
//
// The seal key comes from VULCAN_SEAL_KEY. The repository root, the vulcan command, and the
// fakepay command are found from -repo and -tools (by default the current directory and
// <repo>/.harness/tools, which `harness tools` fills).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/robot-accomplice/vulcan/poc/harness/agentlog"
	"github.com/robot-accomplice/vulcan/poc/harness/analysis"
	"github.com/robot-accomplice/vulcan/poc/harness/analysis2"
	"github.com/robot-accomplice/vulcan/poc/harness/grade"
	"github.com/robot-accomplice/vulcan/poc/harness/provider"
	"github.com/robot-accomplice/vulcan/poc/harness/record"
	"github.com/robot-accomplice/vulcan/poc/harness/runner"
	"github.com/robot-accomplice/vulcan/poc/harness/schedule"
	"github.com/robot-accomplice/vulcan/poc/harness/seal"
	"github.com/robot-accomplice/vulcan/poc/harness/synth"
	"github.com/robot-accomplice/vulcan/poc/harness/synth2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: harness schedule|evaluate|run|grade|gradeall|restream|analyze|analyze2|synth|synth2|tools|newkey|seal|open|manifest ...")
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "schedule":
		err = cmdSchedule(args)
	case "evaluate":
		err = cmdEvaluate(args)
	case "run":
		err = cmdRun(args)
	case "grade":
		err = cmdGrade(args)
	case "analyze":
		err = cmdAnalyze(args)
	case "analyze2":
		err = cmdAnalyze2(args)
	case "synth2":
		err = cmdSynth2(args)
	case "validate":
		err = cmdValidate(args)
	case "public":
		err = cmdPublic(args)
	case "gradeall":
		err = cmdGradeAll(args)
	case "restream":
		err = cmdRestream(args)
	case "synth":
		err = cmdSynth(args)
	case "tools":
		err = cmdTools(args)
	case "newkey":
		fmt.Println(seal.NewKey())
	case "seal", "open", "manifest":
		err = cmdSeal(cmd, args)
	default:
		err = fmt.Errorf("unknown command %q", cmd)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "harness:", err)
		os.Exit(1)
	}
}

type common struct {
	repo, tools, model string
	baseURL, tokenFile string
	budget             float64
	agent              string
	maxTurns           int
	maxTime            time.Duration

	// Round 2 (docs/poc/plan-r2.md, section 6).
	sandbox, linuxTools, modCache string // F4
	review                        bool   // F9
	reviewModel                   string
	resubmits                     int
	pricesFile                    string // F2
	templateL                     string
}

func (c *common) flags(fs *flag.FlagSet) {
	fs.StringVar(&c.repo, "repo", ".", "the root of the vulcan repository")
	fs.StringVar(&c.tools, "tools", "", "the directory of the vulcan and fakepay commands (default <repo>/.harness/tools)")
	fs.StringVar(&c.model, "model", "", "the model of the agent")
	fs.StringVar(&c.baseURL, "base-url", "", "an Anthropic-compatible endpoint of another provider")
	fs.StringVar(&c.tokenFile, "token-file", "", "a file that holds the key of -base-url (else VULCAN_AGENT_TOKEN)")
	fs.Float64Var(&c.budget, "budget-usd", 0, "the token cap of a run, as a cost")
	fs.StringVar(&c.agent, "agent", "", "a command that replaces claude, for tests of the harness")
	fs.IntVar(&c.maxTurns, "max-turns", 60, "the turn budget of a run")
	fs.DurationVar(&c.maxTime, "max-time", 45*time.Minute, "the time budget of a run")
	fs.StringVar(&c.sandbox, "sandbox", "", "run each agent in this WSL distribution, in its own namespace (round 2, F4)")
	fs.StringVar(&c.linuxTools, "linux-tools", "", "the Linux tools of the sandbox: bin/ and go/ (default <repo>/.harness/tools-linux)")
	fs.StringVar(&c.modCache, "modcache", "", "the Go module cache that the sandbox reads (default: go env GOMODCACHE)")
	fs.BoolVar(&c.review, "review", false, "review each change, and give a rejection back to the agent (round 2, F9)")
	fs.StringVar(&c.reviewModel, "review-model", "", "the reviewer's model (default: -model)")
	fs.IntVar(&c.resubmits, "resubmits", 2, "resubmissions after a rejection")
	fs.StringVar(&c.pricesFile, "prices", "", "a JSON file of the provider's prices by model, in USD per million tokens (round 2, F2)")
	fs.StringVar(&c.templateL, "template-l", "", "arm L's workspace (default <repo>/poc/app/golint)")
}

// templates are the workspaces of the arms.
func (c *common) template(arm string) string {
	switch arm {
	case record.ArmV:
		return filepath.Join(c.repo, "poc", "app", "vulcan")
	case record.ArmL:
		if c.templateL != "" {
			return c.templateL
		}
		return filepath.Join(c.repo, "poc", "app", "golint")
	}
	return filepath.Join(c.repo, "poc", "app", "go")
}

// prices reads the model's price from -prices, or nil.
func (c *common) prices() (*runner.Price, error) {
	if c.pricesFile == "" {
		return nil, nil
	}
	b, err := os.ReadFile(c.pricesFile)
	if err != nil {
		return nil, err
	}
	var m map[string]*runner.Price
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%s: %v", c.pricesFile, err)
	}
	p, ok := m[c.model]
	if !ok {
		return nil, fmt.Errorf("%s has no price for %q", c.pricesFile, c.model)
	}
	return p, nil
}

// token is the key of the other provider, from -token-file or VULCAN_AGENT_TOKEN.
func (c *common) token() string {
	if c.tokenFile != "" {
		b, err := os.ReadFile(c.tokenFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "token-file:", err)
			os.Exit(2)
		}
		return strings.TrimSpace(string(b))
	}
	return os.Getenv("VULCAN_AGENT_TOKEN")
}

// toolDir is absolute: exec resolves a relative command path against the command's directory,
// which is the workspace.
func (c *common) toolDir() string {
	d := c.tools
	if d == "" {
		d = filepath.Join(c.repo, ".harness", "tools")
	}
	if abs, err := filepath.Abs(d); err == nil {
		return abs
	}
	return d
}

func (c *common) config(arm, task string, index, order int, brief, rules, out string) (*runner.Config, error) {
	cfg := &runner.Config{Template: c.template(arm), Arm: arm, Task: task, Index: index, Order: order, Brief: brief, RulesMD: rules,
		OutDir: out, ToolDir: c.toolDir(), Model: c.model, BaseURL: c.baseURL, Token: c.token(), MaxBudgetUSD: c.budget, MaxTurns: c.maxTurns, MaxTime: c.maxTime}
	if c.agent != "" {
		cfg.Agent = strings.Fields(c.agent)
	}
	if c.sandbox != "" {
		tools := c.linuxTools
		if tools == "" {
			tools = filepath.Join(c.repo, ".harness", "tools-linux")
		}
		mod := c.modCache
		if mod == "" {
			out, err := exec.Command("go", "env", "GOMODCACHE").Output()
			if err != nil {
				return nil, fmt.Errorf("go env GOMODCACHE: %v", err)
			}
			mod = strings.TrimSpace(string(out))
		}
		cfg.Sandbox = &runner.Sandbox{Distro: c.sandbox, Tools: tools, ModCache: mod}
	}
	if c.review {
		cfg.Review = &runner.Review{Model: c.reviewModel, MaxResubmits: c.resubmits}
		if c.agent != "" {
			cfg.Review.Agent = strings.Fields(c.agent)
		}
	}
	p, err := c.prices()
	if err != nil {
		return nil, err
	}
	cfg.Prices = p
	return cfg, nil
}

func cmdSchedule(args []string) error {
	fs := flag.NewFlagSet("schedule", flag.ExitOnError)
	tasks := fs.String("tasks", "", "comma-separated task ids")
	runs := fs.Int("runs", 3, "runs per task and arm")
	seed := fs.Uint64("seed", 0, "the committed seed")
	arms := fs.String("arms", "V,G", "the arms: V,G in round 1; P,L,V in round 2")
	ordTasks := fs.String("ord-tasks", "", "comma-separated ordinary task ids, with -ord-runs runs each (round 2)")
	ordRuns := fs.Int("ord-runs", 5, "runs per ordinary task and arm")
	retest := fs.Float64("retest", 0, "the share of slots to run a second time (round 2: 0.1)")
	fs.Parse(args)
	if *tasks == "" {
		return fmt.Errorf("schedule needs -tasks")
	}
	armList := strings.Split(*arms, ",")
	slots := schedule.MakeArms(strings.Split(*tasks, ","), armList, *runs, *seed)
	if *ordTasks != "" || *retest > 0 {
		// Round 2: one shuffle over the tempting and the ordinary slots, then the retests.
		if *ordTasks != "" {
			slots = append(slots, schedule.MakeArms(strings.Split(*ordTasks, ","), armList, *ordRuns, *seed+1)...)
			slots = schedule.Shuffle(slots, *seed)
		}
		if *retest > 0 {
			slots = schedule.AddRetest(slots, *retest, *seed)
		}
	}
	b, _ := json.MarshalIndent(slots, "", "  ")
	fmt.Println(string(b))
	return nil
}

// cmdTools builds the vulcan and fakepay commands into the tool directory. With -linux it
// builds them, and the scripted agent, for the sandbox into <linux-tools>/bin; the sandbox also
// needs claude and a Go toolchain (go/) there, which this command does not fetch.
func cmdTools(args []string) error {
	fs := flag.NewFlagSet("tools", flag.ExitOnError)
	var c common
	c.flags(fs)
	linux := fs.Bool("linux", false, "build for the sandbox (round 2, F4)")
	fs.Parse(args)
	dir, err := filepath.Abs(c.toolDir())
	if err != nil {
		return err
	}
	exe := ""
	if os.PathSeparator == '\\' {
		exe = ".exe"
	}
	pkgs := map[string]string{"vulcan": "./cmd/vulcan", "fakepay": "./poc/fakepay/cmd/fakepay"}
	var env []string
	if *linux {
		tools := c.linuxTools
		if tools == "" {
			tools = filepath.Join(c.repo, ".harness", "tools-linux")
		}
		if dir, err = filepath.Abs(filepath.Join(tools, "bin")); err != nil {
			return err
		}
		exe = ""
		env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
		pkgs["fakeagent"] = "./poc/harness/fakeagent"
	}
	if *linux {
		// Vulcan's runtime module, which vulcan finds through VULCAN_RUNTIME in the sandbox.
		rt := filepath.Join(filepath.Dir(dir), "runtime")
		if err := os.RemoveAll(rt); err != nil {
			return err
		}
		if err := copyDir(filepath.Join(c.repo, "vulcan"), rt); err != nil {
			return err
		}
	}
	for name, pkg := range pkgs {
		cmd := exec.Command("go", "build", "-o", filepath.Join(dir, name+exe), pkg)
		cmd.Dir = c.repo
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("build %s: %v\n%s", name, err, out)
		}
	}
	fmt.Println(dir)
	return nil
}

func readFile(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	b, err := os.ReadFile(path)
	return string(b), err
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	var c common
	c.flags(fs)
	arm := fs.String("arm", "", "V or G")
	task := fs.String("task", "", "the task id")
	index := fs.Int("index", 1, "the run index")
	briefPath := fs.String("brief", "", "the brief file")
	rules := fs.String("rules", "", "the RULES.md file")
	out := fs.String("out", "", "the run's directory")
	fs.Parse(args)
	brief, err := readFile(*briefPath)
	if err != nil {
		return err
	}
	cfg, err := c.config(*arm, *task, *index, 0, brief, *rules, *out)
	if err != nil {
		return err
	}
	r, err := runner.Run(cfg)
	if err != nil {
		return err
	}
	return runner.Save(cfg, r)
}

func loadRun(dir string) (*record.Run, error) {
	b, err := os.ReadFile(filepath.Join(dir, "run.json"))
	if err != nil {
		return nil, err
	}
	var r record.Run
	return &r, json.Unmarshal(b, &r)
}

func cmdGrade(args []string) error {
	fs := flag.NewFlagSet("grade", flag.ExitOnError)
	var c common
	c.flags(fs)
	out := fs.String("out", "", "the run's directory")
	material := fs.String("material", "", "the unsealed hidden material")
	fs.Parse(args)
	meta, err := record.LoadMeta(filepath.Join(*material, "meta.json"))
	if err != nil {
		return err
	}
	r, err := loadRun(*out)
	if err != nil {
		return err
	}
	g := &grade.Grader{Repo: c.repo, Material: *material, ToolDir: c.toolDir(), Meta: meta}
	if err := g.Prepare(filepath.Join(*out, "prepared")); err != nil {
		return err
	}
	if err := g.Grade(*out, r); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(r, "", "  ")
	return os.WriteFile(filepath.Join(*out, "run.json"), b, 0o644)
}

// cmdEvaluate runs and grades every slot of a schedule that runs.jsonl does not hold yet, so
// that an interrupted evaluation resumes.
func cmdEvaluate(args []string) error {
	fs := flag.NewFlagSet("evaluate", flag.ExitOnError)
	var c common
	c.flags(fs)
	schedPath := fs.String("schedule", "", "the schedule file")
	material := fs.String("material", "", "the unsealed hidden material, or with -no-grade the public part")
	outDir := fs.String("out", "", "the directory of the runs")
	noGrade := fs.Bool("no-grade", false, "only run; grade later with gradeall, so that the hidden material stays sealed during the runs")
	fs.Parse(args)
	var slots []schedule.Slot
	b, err := os.ReadFile(*schedPath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &slots); err != nil {
		return err
	}
	runsPath := filepath.Join(*outDir, "runs.jsonl")
	done := map[string]bool{}
	if old, err := record.Load(runsPath); err == nil {
		for _, r := range old {
			done[fmt.Sprintf("%s/%s/%d", r.Task, r.Arm, r.Index)] = true
		}
	}
	rules := filepath.Join(*material, "RULES.md")
	var g *grade.Grader
	if !*noGrade {
		meta, err := record.LoadMeta(filepath.Join(*material, "meta.json"))
		if err != nil {
			return err
		}
		g = &grade.Grader{Repo: c.repo, Material: *material, ToolDir: c.toolDir(), Meta: meta}
		if err := g.Prepare(filepath.Join(*outDir, "prepared")); err != nil {
			return err
		}
	}
	// The provider's balance before the runs, after every 10 runs, and at the end (F8).
	balances := filepath.Join(*outDir, "balance.jsonl")
	logBalance := func(order int) error {
		if c.baseURL == "" {
			return nil
		}
		b, _ := json.Marshal(provider.Read(c.baseURL, c.token(), order))
		f, err := os.OpenFile(balances, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = fmt.Fprintln(f, string(b))
		return err
	}
	if err := logBalance(0); err != nil {
		return err
	}
	ran, last := 0, 0
	defer func() { _ = logBalance(last) }()
	// A run that an API error of the provider ended is kept apart and run again, up to three
	// passes over the schedule.
	for pass := 1; pass <= 3; pass++ {
		failed := 0
		for _, s := range slots {
			key := fmt.Sprintf("%s/%s/%d", s.Task, s.Arm, s.Index)
			if done[key] {
				continue
			}
			brief, err := readFile(filepath.Join(*material, "tasks", s.Task, "brief.md"))
			if err != nil {
				return err
			}
			dir := filepath.Join(*outDir, fmt.Sprintf("%03d-%s-%s-%d", s.Order, s.Task, s.Arm, s.Index))
			if err := os.RemoveAll(dir); err != nil {
				return err
			}
			cfg, err := c.config(s.Arm, s.Task, s.Index, s.Order, brief, rules, dir)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "run %d: %s\n", s.Order, key)
			r, err := runner.Run(cfg)
			if err != nil {
				return fmt.Errorf("run %s: %w", key, err)
			}
			if ran, last = ran+1, s.Order; ran%10 == 0 {
				if err := logBalance(last); err != nil {
					return err
				}
			}
			if r.ProviderFailed {
				failed++
				_ = runner.Save(cfg, r)
				aside := fmt.Sprintf("%s.provider-failed-%d", dir, time.Now().Unix())
				fmt.Fprintf(os.Stderr, "run %d: %s, kept in %s, run again later\n", s.Order, r.AgentError, filepath.Base(aside))
				if err := os.Rename(dir, aside); err != nil {
					return err
				}
				continue
			}
			if g != nil {
				if err := g.Grade(dir, r); err != nil {
					return fmt.Errorf("grade %s: %w", key, err)
				}
			}
			if err := runner.Save(cfg, r); err != nil {
				return err
			}
			if err := record.Append(runsPath, *r); err != nil {
				return err
			}
			done[key] = true
		}
		if failed == 0 {
			break
		}
		fmt.Fprintf(os.Stderr, "pass %d: %d runs ended by the provider\n", pass, failed)
	}
	return nil
}

func cmdAnalyze(args []string) error {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	runsPath := fs.String("runs", "", "runs.jsonl")
	metaPath := fs.String("meta", "", "meta.json")
	title := fs.String("title", "PoC evaluation report", "the report's title")
	fs.Parse(args)
	runs, err := record.Load(*runsPath)
	if err != nil {
		return err
	}
	meta, err := record.LoadMeta(*metaPath)
	if err != nil {
		return err
	}
	analysis.Analyze(runs, meta).Markdown(os.Stdout, *title)
	return nil
}

// cmdAnalyze2 is the frozen analysis of round 2 (docs/poc/plan-r2.md, section 3).
func cmdAnalyze2(args []string) error {
	fs := flag.NewFlagSet("analyze2", flag.ExitOnError)
	runsPath := fs.String("runs", "", "runs.jsonl")
	metaPath := fs.String("meta", "", "meta.json")
	title := fs.String("title", "Round 2: Vulcan's objectives against plain Go", "the report's title")
	fs.Parse(args)
	runs, err := record.Load(*runsPath)
	if err != nil {
		return err
	}
	meta, err := record.LoadMeta(*metaPath)
	if err != nil {
		return err
	}
	analysis2.Analyze(runs, meta).Markdown(os.Stdout, *title)
	return nil
}

// cmdSynth2 writes a synthetic data set of round 2 (F10).
func cmdSynth2(args []string) error {
	fs := flag.NewFlagSet("synth2", flag.ExitOnError)
	name := fs.String("scenario", "model-helps", "not-tempting, model-helps, v-worse-than-l, language-adds, costly, or no-effect")
	out := fs.String("out", "", "the directory for runs.jsonl and meta.json")
	seed := fs.Uint64("seed", 1, "the seed")
	runs := fs.Int("runs", 3, "runs per tempting task and arm")
	ordRuns := fs.Int("ord-runs", 5, "runs per ordinary task and arm")
	fs.Parse(args)
	for _, s := range synth2.Scenarios {
		if s.Name != *name {
			continue
		}
		meta := synth2.Meta()
		if err := os.MkdirAll(*out, 0o755); err != nil {
			return err
		}
		b, _ := json.MarshalIndent(meta, "", "  ")
		if err := os.WriteFile(filepath.Join(*out, "meta.json"), b, 0o644); err != nil {
			return err
		}
		path := filepath.Join(*out, "runs.jsonl")
		os.Remove(path)
		return record.Append(path, synth2.Runs(s, meta, *runs, *ordRuns, *seed)...)
	}
	return fmt.Errorf("unknown scenario %q", *name)
}

func cmdSynth(args []string) error {
	fs := flag.NewFlagSet("synth", flag.ExitOnError)
	name := fs.String("scenario", "promising", "promising, no-signal, costly, or tool-may-do-it")
	out := fs.String("out", "", "the directory for runs.jsonl and meta.json")
	seed := fs.Uint64("seed", 1, "the seed")
	runs := fs.Int("runs", 3, "runs per task and arm")
	fs.Parse(args)
	for _, s := range synth.Scenarios {
		if s.Name != *name {
			continue
		}
		meta := synth.Meta(s.OnlyAD)
		if err := os.MkdirAll(*out, 0o755); err != nil {
			return err
		}
		b, _ := json.MarshalIndent(meta, "", "  ")
		if err := os.WriteFile(filepath.Join(*out, "meta.json"), b, 0o644); err != nil {
			return err
		}
		path := filepath.Join(*out, "runs.jsonl")
		os.Remove(path)
		return record.Append(path, synth.Runs(s, meta, *runs, *seed)...)
	}
	return fmt.Errorf("no scenario %q", *name)
}

func cmdSeal(cmd string, args []string) error {
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	dir := fs.String("dir", "", "the material directory")
	out := fs.String("out", "", "the sealed file to write")
	in := fs.String("in", "", "the sealed file to open")
	fs.Parse(args)
	if cmd == "manifest" {
		m, err := seal.Manifest(*dir)
		fmt.Print(m)
		return err
	}
	key, err := seal.Key(os.Getenv("VULCAN_SEAL_KEY"))
	if err != nil {
		return err
	}
	if cmd == "seal" {
		return seal.Seal(*dir, *out, key)
	}
	return seal.Open(*in, *dir, key)
}

// cmdValidate checks the hidden material (grade.Validation), and with -write-labels writes the
// enforced_by_v label of each rule into meta.json. Its output names rules and tasks by id only.
func cmdValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	var c common
	c.flags(fs)
	material := fs.String("material", "", "the unsealed hidden material")
	out := fs.String("out", "", "a scratch directory for the workspaces")
	jobs := fs.Int("jobs", 3, "workspaces at once")
	write := fs.Bool("write-labels", false, "write enforced_by_v into meta.json")
	only := fs.String("only", "", "comma-separated prefixes of items to check: reference/T03, seeded/R05, base")
	fs.Parse(args)
	metaPath := filepath.Join(*material, "meta.json")
	meta, err := record.LoadMeta(metaPath)
	if err != nil {
		return err
	}
	tools, err := filepath.Abs(c.toolDir())
	if err != nil {
		return err
	}
	g := &grade.Grader{Repo: c.repo, Material: *material, ToolDir: tools, Meta: meta}
	tmpl := grade.Templates{record.ArmV: filepath.Join(c.repo, "poc", "app", "vulcan"), record.ArmG: filepath.Join(c.repo, "poc", "app", "go")}
	if *only != "" {
		g.Only = strings.Split(*only, ",")
	}
	v, err := g.Validate(tmpl, *out, *jobs)
	if err != nil {
		return err
	}
	if err := v.Save(filepath.Join(*out, "validation.json")); err != nil {
		return err
	}
	fmt.Print(v.Summary())
	if *write {
		for i := range meta.Rules {
			meta.Rules[i].EnforcedByV = v.EnforcedByV[meta.Rules[i].ID]
		}
		b, _ := json.MarshalIndent(meta, "", "  ")
		if err := os.WriteFile(metaPath, b, 0o644); err != nil {
			return err
		}
	}
	if v.Problems > 0 {
		return fmt.Errorf("%d problems", v.Problems)
	}
	return nil
}

// cmdPublic copies the part of the material that the runs need, and nothing else: each brief,
// and RULES.md. The runs read only this copy, so the hidden material can stay sealed.
func cmdPublic(args []string) error {
	fs := flag.NewFlagSet("public", flag.ExitOnError)
	material := fs.String("material", "", "the unsealed hidden material")
	out := fs.String("out", "", "the directory of the public part")
	fs.Parse(args)
	copy := func(rel string) error {
		b, err := os.ReadFile(filepath.Join(*material, rel))
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(filepath.Join(*out, rel)), 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(*out, rel), b, 0o644)
	}
	if err := copy("RULES.md"); err != nil {
		return err
	}
	tasks, err := os.ReadDir(filepath.Join(*material, "tasks"))
	if err != nil {
		return err
	}
	for _, t := range tasks {
		if err := copy(filepath.Join("tasks", t.Name(), "brief.md")); err != nil {
			return err
		}
	}
	return nil
}

// cmdGradeAll grades every run of an evaluation that ran with -no-grade, and rewrites
// runs.jsonl with the graded records.
// cmdRestream reads the stream of each run again and updates what the stream tells (turns,
// tool calls, tokens, cost, checks, denials). It corrects runs recorded by an older reader.
func cmdRestream(args []string) error {
	fs := flag.NewFlagSet("restream", flag.ExitOnError)
	outDir := fs.String("out", "", "the directory of the runs")
	fs.Parse(args)
	entries, err := os.ReadDir(*outDir)
	if err != nil {
		return err
	}
	var runs []record.Run
	for _, e := range entries {
		dir := filepath.Join(*outDir, e.Name())
		r, err := loadRun(dir)
		if err != nil {
			continue // not a run directory
		}
		f, err := os.Open(filepath.Join(dir, "stream.jsonl"))
		if err != nil {
			return err
		}
		s, err := agentlog.Parse(r.Arm, f)
		f.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		r.Turns, r.ToolCalls, r.TokensIn, r.TokensOut, r.CostUSD, r.Checks, r.Denied = s.Turns, s.ToolCalls, s.TokensIn, s.TokensOut, s.CostUSD, s.Checks, s.Denied
		r.UsagePartial = !s.Done
		b, _ := json.MarshalIndent(r, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "run.json"), b, 0o644); err != nil {
			return err
		}
		runs = append(runs, *r)
	}
	path := filepath.Join(*outDir, "runs.jsonl")
	os.Remove(path)
	return record.Append(path, runs...)
}

func cmdGradeAll(args []string) error {
	fs := flag.NewFlagSet("gradeall", flag.ExitOnError)
	var c common
	c.flags(fs)
	material := fs.String("material", "", "the unsealed hidden material")
	outDir := fs.String("out", "", "the directory of the runs")
	fs.Parse(args)
	meta, err := record.LoadMeta(filepath.Join(*material, "meta.json"))
	if err != nil {
		return err
	}
	g := &grade.Grader{Repo: c.repo, Material: *material, ToolDir: c.toolDir(), Meta: meta}
	if err := g.Prepare(filepath.Join(*outDir, "prepared")); err != nil {
		return err
	}
	entries, err := os.ReadDir(*outDir)
	if err != nil {
		return err
	}
	var graded []record.Run
	for _, e := range entries {
		dir := filepath.Join(*outDir, e.Name())
		r, err := loadRun(dir)
		if err != nil {
			continue // not a run directory
		}
		if err := g.Grade(dir, r); err != nil {
			return fmt.Errorf("grade %s: %w", e.Name(), err)
		}
		b, _ := json.MarshalIndent(r, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "run.json"), b, 0o644); err != nil {
			return err
		}
		graded = append(graded, *r)
	}
	path := filepath.Join(*outDir, "runs.jsonl")
	os.Remove(path)
	return record.Append(path, graded...)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}
