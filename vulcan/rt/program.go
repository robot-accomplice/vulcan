package rt

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// FlagSpec describes one flag that a container reads at startup (spec §5.3).
type FlagSpec struct {
	Name       string
	Type       string // "string", "int", "int64", "float64", "bool", "duration"
	Default    any    // of the flag's type
	HasDefault bool
}

// Flags holds the parsed values of a container's flags.
type Flags struct {
	values map[string]any
}

func (f *Flags) String(name string) string          { return f.values[name].(string) }
func (f *Flags) Int(name string) int                { return f.values[name].(int) }
func (f *Flags) Int64(name string) int64            { return f.values[name].(int64) }
func (f *Flags) Float64(name string) float64        { return f.values[name].(float64) }
func (f *Flags) Bool(name string) bool              { return f.values[name].(bool) }
func (f *Flags) Duration(name string) time.Duration { return f.values[name].(time.Duration) }

// EnvName gives the environment variable that a flag reads (spec §5.3).
func EnvName(flag string) string {
	return strings.NewReplacer("-", "_", ".", "_").Replace(strings.ToUpper(flag))
}

// ParseFlags reads each flag from the command line, then from the environment, then from its
// default. It returns one error that lists every missing or invalid flag.
func ParseFlags(specs []FlagSpec, args []string) (*Flags, error) {
	given := map[string]string{}
	var problems []string
	for _, a := range args {
		if !strings.HasPrefix(a, "--") {
			problems = append(problems, fmt.Sprintf("unexpected argument %q", a))
			continue
		}
		name, val, ok := strings.Cut(strings.TrimPrefix(a, "--"), "=")
		if !ok {
			val = "true"
		}
		given[name] = val
	}
	known := map[string]bool{}
	f := &Flags{values: map[string]any{}}
	for _, s := range specs {
		known[s.Name] = true
		raw, ok := given[s.Name]
		if !ok {
			raw, ok = os.LookupEnv(EnvName(s.Name))
		}
		if !ok {
			if !s.HasDefault {
				problems = append(problems, fmt.Sprintf("flag --%s (or %s) is required", s.Name, EnvName(s.Name)))
				continue
			}
			f.values[s.Name] = s.Default
			continue
		}
		v, err := parseValue(s.Type, raw)
		if err != nil {
			problems = append(problems, fmt.Sprintf("flag --%s: %v", s.Name, err))
			continue
		}
		f.values[s.Name] = v
	}
	for name := range given {
		if !known[name] {
			problems = append(problems, fmt.Sprintf("unknown flag --%s", name))
		}
	}
	if len(problems) > 0 {
		return nil, errors.New(strings.Join(problems, "\n"))
	}
	return f, nil
}

func parseValue(typ, raw string) (any, error) {
	switch typ {
	case "string":
		return raw, nil
	case "int":
		return strconv.Atoi(raw)
	case "int64":
		return strconv.ParseInt(raw, 10, 64)
	case "float64":
		return strconv.ParseFloat(raw, 64)
	case "bool":
		return strconv.ParseBool(raw)
	case "duration":
		return time.ParseDuration(raw)
	}
	return nil, fmt.Errorf("unsupported flag type %s", typ)
}

// Step is a setup or startup function of one leaf.
type Step struct {
	Leaf string
	Fn   func(ctx context.Context) error
}

// Program is a container as the generated main describes it (spec §6).
type Program struct {
	Name            string
	ShutdownTimeout time.Duration
	Flags           []FlagSpec
	// Construct calls each constructor in wiring order and binds each port.
	Construct func(f *Flags) error
	// Setups and Startups are filled by Construct's caller, after Construct returns.
	Setups   func(f *Flags) []Step
	Startups func(f *Flags) []Step
	// OnContractFailure is "panic" or "exit" (config on_contract_failure, §7.4).
	OnContractFailure string
}

// Main runs a program and returns the process exit code.
func Main(p Program) int {
	report := func(stage string, err error) int {
		fmt.Fprintf(os.Stderr, "%s: %s failed:\n%v\n", p.Name, stage, err)
		return 1
	}
	// --on-contract-failure=panic|exit overrides the config key (§7.4).
	mode := p.OnContractFailure
	var args []string
	for _, a := range os.Args[1:] {
		if v, ok := strings.CutPrefix(a, "--on-contract-failure="); ok {
			mode = v
			continue
		}
		args = append(args, a)
	}
	switch mode {
	case "exit":
		ExitOnFailure = true
	case "", "panic":
	default:
		return report("reading flags", fmt.Errorf("--on-contract-failure is panic or exit, not %q", mode))
	}
	flags, err := ParseFlags(p.Flags, args)
	if err != nil {
		return report("reading flags", err)
	}
	if err := guard("construct", p.Construct, flags); err != nil {
		return report("startup", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runAll(ctx, p.Setups(flags)); err != nil {
		return report("setup", err)
	}
	startCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	steps := p.Startups(flags)
	done := make(chan error, len(steps))
	for _, s := range steps {
		go func() { done <- runStep(startCtx, s) }()
	}
	timeout := p.ShutdownTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	// stopRest cancels the startup functions that still run, and waits for them for at most the
	// shutdown timeout (§6, stop steps 1 to 3).
	stopRest := func(remaining int) {
		cancel()
		deadline := time.After(timeout)
		for ; remaining > 0; remaining-- {
			select {
			case <-done:
			case <-deadline:
				fmt.Fprintf(os.Stderr, "%s: %d startup functions did not stop within %s\n", p.Name, remaining, timeout)
				return
			}
		}
	}
	for remaining := len(steps); remaining > 0; {
		select {
		case err := <-done:
			remaining--
			if err != nil {
				// A failed startup function: report at once, stop the rest, exit 1.
				code := report("startup", err)
				stopRest(remaining)
				return code
			}
		case <-ctx.Done():
			// A stop signal: exit 0, also when a startup function does not stop in time.
			stopRest(remaining)
			return 0
		}
	}
	return 0
}

func runAll(ctx context.Context, steps []Step) error {
	var wg sync.WaitGroup
	errs := make([]error, len(steps))
	for i, s := range steps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = runStep(ctx, s)
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}

func runStep(ctx context.Context, s Step) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s: panic: %v", s.Leaf, r)
		}
	}()
	if err := s.Fn(ctx); err != nil {
		return fmt.Errorf("%s: %w", s.Leaf, err)
	}
	return nil
}

func guard(what string, fn func(*Flags) error, f *Flags) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s: panic: %v", what, r)
		}
	}()
	return fn(f)
}

// Constructed wraps the error of a failed constructor with the constructor's name.
func Constructed(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("constructor %s: %w", name, err)
}
