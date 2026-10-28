package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Sandbox runs the agent in WSL, in its own user and mount namespace (round 2, F4), so that no
// run sees another run's files. It needs no root: `unshare --map-root-user` gives the run its
// own root inside the namespace, with the rights of the user outside it. Inside:
//
//   - /tmp is a fresh tmpfs; the run's directory is at /tmp/run (workspace, scratch, home);
//   - the tools are at /tmp/tools, read-only: bin/ (claude, vulcan, fakepay, vulcan-lint) and
//     go/ (the Go toolchain);
//   - Go modules come from a read-only file proxy of the host's module cache, so no module comes
//     from the network;
//   - the user's home, /mnt (the Windows drives), /var/tmp, and /dev/shm are hidden by empty
//     tmpfs mounts.
//
// The network stays open: the agent needs the provider. The host's /tmp leftovers (round 1,
// fault 4) cannot happen here.
type Sandbox struct {
	Distro   string // the WSL distribution, such as Ubuntu-22.04
	Tools    string // the host directory of the Linux tools (bin/ and go/)
	ModCache string // the host's GOMODCACHE
}

// sandboxRun is the run's directory inside the sandbox.
const sandboxRun = "/tmp/run"

// tokenVar carries the provider's key into WSL through WSLENV, so that it is in no file.
const tokenVar = "VULCAN_SB_TOKEN"

// wslPath gives the WSL path of a host path: C:\a\b is /mnt/c/a/b.
func wslPath(p string) string {
	abs, err := filepath.Abs(p)
	if err == nil {
		p = abs
	}
	p = filepath.ToSlash(p)
	if len(p) >= 2 && p[1] == ':' {
		return "/mnt/" + strings.ToLower(p[:1]) + p[2:]
	}
	return p
}

// shQuote quotes a word for sh.
func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// script is the shell script that sets up the namespace and starts the agent.
func (sb *Sandbox) script(c *Config, args []string) string {
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	w("#!/bin/sh")
	w("# Made by the harness for one run (poc/harness/runner/sandbox.go). It holds no key.")
	w("set -eu")
	w("real_home=\"$HOME\"")
	w("mount -t tmpfs tmpfs /tmp")
	w("mkdir -p %s /tmp/tools /tmp/modproxy /tmp/go /tmp/t", sandboxRun)
	w("mount --bind %s %s", shQuote(wslPath(c.OutDir)), sandboxRun)
	w("mount --bind %s /tmp/tools && mount -o remount,bind,ro /tmp/tools", shQuote(wslPath(sb.Tools)))
	w("mount --bind %s /tmp/modproxy && mount -o remount,bind,ro /tmp/modproxy", shQuote(wslPath(filepath.Join(sb.ModCache, "cache", "download"))))
	w("mount -t tmpfs tmpfs \"$real_home\"")
	w("mount -t tmpfs tmpfs /mnt")
	w("mount -t tmpfs tmpfs /var/tmp")
	w("mount -t tmpfs tmpfs /dev/shm")
	w("cd %s/workspace", sandboxRun)
	env := []string{
		"HOME=" + sandboxRun + "/home",
		"PATH=/tmp/tools/bin:/tmp/tools/go/bin:/usr/local/bin:/usr/bin:/bin",
		"TMPDIR=/tmp/t",
		"GOTOOLCHAIN=local", "GOFLAGS=-mod=mod", "GOPROXY=file:///tmp/modproxy", "GOSUMDB=off",
		"GOMODCACHE=/tmp/go/mod", "GOCACHE=/tmp/go/cache", "GOPATH=/tmp/go/path",
	}
	env = append(env, c.providerEnv()...)
	for _, e := range env {
		k, v, _ := strings.Cut(e, "=")
		w("export %s=%s", k, shQuote(v))
	}
	if c.BaseURL != "" {
		w("export ANTHROPIC_AUTH_TOKEN=\"${%s:-}\"", tokenVar)
	}
	w("unset %s", tokenVar)
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = shQuote(a)
	}
	w("exec %s", strings.Join(quoted, " "))
	return b.String()
}

// command makes the WSL process of one agent round. Stopping it kills the namespace: unshare
// --kill-child takes the namespace's first process, and with it every process of the run.
func (sb *Sandbox) command(ctx context.Context, c *Config, args []string) (*exec.Cmd, func(), error) {
	path := filepath.Join(c.OutDir, "sandbox.sh")
	if err := os.WriteFile(path, []byte(sb.script(c, args)), 0o644); err != nil {
		return nil, nil, err
	}
	marker := wslPath(path)
	cmd := exec.CommandContext(ctx, "wsl.exe", "-d", sb.Distro, "--", "unshare", "--user", "--map-root-user",
		"--mount", "--pid", "--fork", "--kill-child", "--mount-proc", "sh", marker)
	var env []string
	wslenv := tokenVar
	for _, e := range os.Environ() {
		k, v, _ := strings.Cut(e, "=")
		switch {
		case strings.EqualFold(k, "WSLENV"):
			if v != "" {
				wslenv = v + ":" + tokenVar
			}
		case !stripped(k):
			env = append(env, e)
		}
	}
	cmd.Env = append(env, "WSLENV="+wslenv, tokenVar+"="+c.Token)
	kill := func() {
		_ = exec.Command("wsl.exe", "-d", sb.Distro, "--", "pkill", "-KILL", "-f", marker).Run()
	}
	cmd.Cancel = func() error {
		kill()
		return killTree(cmd.Process)
	}
	cmd.WaitDelay = 10 * time.Second
	// A round that ends by itself leaves nothing running; kill what a background shell may have
	// left, so that no process outlives its run.
	return cmd, kill, nil
}
