// Package project loads a Vulcan project: its go.mod, system.vulcan, leaves, libraries, and code
// files, and reports the layout diagnostics (spec §5.2, Appendix B).
package project

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/robot-accomplice/vulcan/internal/blank"
	"github.com/robot-accomplice/vulcan/internal/decl"
	"github.com/robot-accomplice/vulcan/internal/diag"
	"github.com/robot-accomplice/vulcan/internal/lex"
)

// Release names the release in K0009 messages.
const Release = "the Vulcan PoC (release 1 subset)"

// Source is one .vulcan file.
type Source struct {
	Rel   string // relative to the root, with slashes
	Src   []byte
	lines *lex.Lines
}

// Pos converts an offset to a diagnostic position.
func (s *Source) Pos(off int) diag.Pos {
	if s.lines == nil {
		s.lines = lex.NewLines(s.Src)
	}
	l, c := s.lines.Pos(off)
	return diag.Pos{File: s.Rel, Line: l, Col: c}
}

// CodeFile is a code or test file of a leaf or a library.
type CodeFile struct {
	*Source
	Test  bool
	Blank *blank.Result
}

// Package is a leaf or a library: one directory, one Go package.
type Package struct {
	Dir        string // relative, with slashes
	ImportPath string
	GoName     string // the package name: the directory name
	Files      []*CodeFile
	IsLeaf     bool

	// Leaves only.
	Name      string // the component name
	Container string
	Entry     *decl.LeafEntry
	Comp      *decl.Component
	CompSrc   *Source
	Gateway   bool
	EntryLeaf bool

	// Libraries only.
	Lib    *decl.Library
	LibSrc *Source
}

// Project is a loaded project.
type Project struct {
	Root      string
	Module    string
	GoMod     []byte
	GoSum     []byte
	SysSrc    *Source
	System    *decl.System
	Container *decl.Container // the one container that release 1 supports
	Leaves    []*Package
	Libs      []*Package
	ByName    map[string]*Package // leaves by component name
	ByPath    map[string]*Package // leaves and libraries by import path
	Diags     *diag.List
}

// Deferred reports a construct that the release does not support (K0009).
func (p *Project) Deferred(pos diag.Pos, what, section string) {
	p.Diags.Errorf(pos, "K0009", "%s is not supported in %s (spec %s)", what, Release, section)
}

// Load reads the project at root.
func Load(root string) (*Project, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	p := &Project{Root: root, Diags: &diag.List{}, ByName: map[string]*Package{}, ByPath: map[string]*Package{}}
	p.GoMod, err = os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, fmt.Errorf("no go.mod in %s: a Vulcan project is a Go module", root)
	}
	p.GoSum, _ = os.ReadFile(filepath.Join(root, "go.sum"))
	p.Module = modulePath(p.GoMod)
	if p.Module == "" {
		return nil, fmt.Errorf("go.mod in %s has no module line", root)
	}
	sysBytes, err := os.ReadFile(filepath.Join(root, "system.vulcan"))
	if err != nil {
		return nil, fmt.Errorf("no system.vulcan in %s", root)
	}
	p.SysSrc = &Source{Rel: "system.vulcan", Src: sysBytes}
	sys, probs := decl.ParseSystem(sysBytes)
	p.problems(p.SysSrc, probs)
	p.System = sys
	if err := p.scan(); err != nil {
		return nil, err
	}
	p.checkSystem()
	return p, nil
}

func modulePath(gomod []byte) string {
	for _, line := range strings.Split(string(gomod), "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && f[0] == "module" {
			return strings.Trim(f[1], `"`)
		}
	}
	return ""
}

func (p *Project) problems(s *Source, probs []decl.Problem) {
	for _, pr := range probs {
		p.Diags.Errorf(s.Pos(pr.Off), pr.Code, "%s", pr.Msg)
	}
}

// scan walks the directories and loads each leaf and library.
func (p *Project) scan() error {
	dirs := map[string][]string{} // rel dir -> .vulcan file names
	err := filepath.WalkDir(p.Root, func(full string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if full != p.Root && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "testdata" || name == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".vulcan") {
			return nil
		}
		rel, _ := filepath.Rel(p.Root, filepath.Dir(full))
		rel = filepath.ToSlash(rel)
		dirs[rel] = append(dirs[rel], name)
		return nil
	})
	if err != nil {
		return err
	}
	var keys []string
	for k := range dirs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var leafDirs []string
	for _, dir := range keys {
		names := dirs[dir]
		sort.Strings(names)
		has := func(n string) bool {
			for _, x := range names {
				if x == n {
					return true
				}
			}
			return false
		}
		if dir == "." {
			for _, n := range names {
				if n != "system.vulcan" {
					p.Diags.Errorf(diag.Pos{File: n}, "K0002", "a code file outside a leaf or a library: the root holds only system.vulcan")
				}
			}
			continue
		}
		isLeaf, isLib := has("component.vulcan"), has("library.vulcan")
		for _, ld := range leafDirs {
			if strings.HasPrefix(dir, ld+"/") {
				p.Diags.Errorf(diag.Pos{File: dir + "/" + names[0]}, "K0002", "directory %s is inside leaf directory %s: a leaf holds no child leaves or libraries", dir, ld)
			}
		}
		switch {
		case isLeaf && isLib:
			p.Diags.Errorf(diag.Pos{File: dir + "/library.vulcan"}, "K0002", "directory %s has both component.vulcan and library.vulcan", dir)
		case isLeaf:
			leafDirs = append(leafDirs, dir)
			if err := p.loadPackage(dir, names, true); err != nil {
				return err
			}
		case isLib:
			if err := p.loadPackage(dir, names, false); err != nil {
				return err
			}
		default:
			for _, n := range names {
				p.Diags.Errorf(diag.Pos{File: dir + "/" + n}, "K0002", "a code file outside a leaf or a library: directory %s has no component.vulcan or library.vulcan", dir)
			}
		}
	}
	return nil
}

func (p *Project) loadPackage(dir string, names []string, leaf bool) error {
	pkg := &Package{Dir: dir, ImportPath: p.Module + "/" + dir, GoName: path.Base(dir), IsLeaf: leaf}
	for _, n := range names {
		b, err := os.ReadFile(filepath.Join(p.Root, filepath.FromSlash(dir), n))
		if err != nil {
			return err
		}
		src := &Source{Rel: dir + "/" + n, Src: b}
		switch n {
		case "component.vulcan":
			c, probs := decl.ParseComponent(b)
			p.problems(src, probs)
			pkg.Comp, pkg.CompSrc, pkg.Name = c, src, c.Name
			pkg.Gateway = len(c.Gateways) > 0
			pkg.EntryLeaf = len(c.Entries) > 0
			for _, d := range c.Deferred {
				p.Deferred(src.Pos(d.Pos), d.What, d.Section)
			}
			if c.Name != "" && !strings.EqualFold(c.Name, pkg.GoName) {
				p.Diags.Errorf(src.Pos(c.Pos), "K0004", "component %s is in directory %s; the name must match the directory, ignoring case", c.Name, dir)
			}
		case "library.vulcan":
			l, probs := decl.ParseLibrary(b)
			p.problems(src, probs)
			pkg.Lib, pkg.LibSrc = l, src
			if l.Name != "" && l.Name != pkg.GoName {
				p.Diags.Errorf(src.Pos(l.Pos), "K0004", "library %s is in directory %s; a library name is its package name, the directory name", l.Name, dir)
			}
		default:
			cf := &CodeFile{Source: src, Test: strings.HasSuffix(n, "_test.vulcan")}
			cf.Blank = blank.File(b)
			p.codeFileDiags(pkg, cf)
			pkg.Files = append(pkg.Files, cf)
		}
	}
	if leaf {
		p.Leaves = append(p.Leaves, pkg)
	} else {
		p.Libs = append(p.Libs, pkg)
	}
	p.ByPath[pkg.ImportPath] = pkg
	return nil
}

func (p *Project) codeFileDiags(pkg *Package, cf *CodeFile) {
	r := cf.Blank
	for _, e := range r.LexErrors {
		p.Diags.Errorf(cf.Pos(e.Off), "K0001", "%s", e.Msg)
	}
	for _, d := range r.Deferred {
		p.Deferred(cf.Pos(d.Off), d.What, d.Section)
	}
	for _, off := range r.Linknames {
		p.Diags.Errorf(cf.Pos(off), "K0122", "a //go:linkname directive in a leaf or a library hides a reference that rule 1 cannot see (§5.4 rule 13)")
	}
	for _, c := range r.Misplaced {
		p.Diags.Errorf(cf.Pos(c.WordOff), "K0513", "an invariant must be on a struct type declared at package level, outside a type ( ) group")
	}
	// Package clause.
	toks, _ := lex.Scan(r.Go)
	for i, t := range toks {
		if t.Tok.String() == "package" && i+1 < len(toks) {
			name := toks[i+1].Lit
			want := pkg.GoName
			if cf.Test && name == want+"_test" {
				p.Diags.Errorf(cf.Pos(toks[i+1].Off), "K0004", "external test packages (%s) are not supported in %s; use package %s", name, Release, want)
			} else if name != want {
				p.Diags.Errorf(cf.Pos(toks[i+1].Off), "K0004", "the package clause names %s; it must name the directory, %s", name, want)
			}
			break
		}
	}
}

// checkSystem checks system.vulcan against the directories.
func (p *Project) checkSystem() {
	sys := p.System
	if sys == nil {
		return
	}
	pos := p.SysSrc.Pos
	if len(sys.Containers) == 0 {
		p.Diags.Errorf(pos(sys.Pos), "K0001", "system %s has no container", sys.Name)
	}
	for i, c := range sys.Containers {
		if i == 0 {
			p.Container = c
			continue
		}
		p.Deferred(pos(c.Pos), "a second container", "§5.5")
	}
	for _, w := range sys.Wires {
		p.Deferred(pos(w.Pos), "a wire between containers", "§5.5")
	}
	if sys.Config != nil {
		p.checkConfig(sys.Config)
	}
	nodes := map[string]string{}
	for _, n := range sys.Persons {
		nodes[n.Name] = "person"
	}
	for _, n := range sys.Externals {
		if nodes[n.Name] != "" {
			p.Diags.Errorf(pos(n.Pos), "K0121", "the name %s is declared twice", n.Name)
		}
		nodes[n.Name] = "external"
	}
	// Leaves in the system, and their directories.
	byDir := map[string]*Package{}
	for _, l := range p.Leaves {
		byDir[l.Dir] = l
	}
	groups := map[string]bool{}
	listed := map[string]bool{}
	for ci, c := range sys.Containers {
		for _, g := range c.Groups {
			groups[g.Name] = true
		}
		for _, le := range c.Leaves {
			if prev, ok := p.ByName[le.Name]; ok {
				_ = prev
				p.Diags.Errorf(pos(le.Pos), "K0121", "two leaves are named %s; leaf names are unique in the system", le.Name)
				continue
			}
			dir := strings.ToLower(c.Name) + "/" + strings.ToLower(le.Name)
			leaf := byDir[dir]
			if leaf == nil {
				if ci == 0 {
					p.Diags.Errorf(pos(le.Pos), "K0003", "leaf %s of container %s has no directory %s with a component.vulcan", le.Name, c.Name, dir)
				}
				continue
			}
			listed[dir] = true
			leaf.Container, leaf.Entry = c.Name, le
			if leaf.Name != "" && leaf.Name != le.Name {
				p.Diags.Errorf(pos(le.Pos), "K0003", "system.vulcan lists %s, but %s/component.vulcan declares component %s", le.Name, dir, leaf.Name)
			}
			leaf.Name = le.Name
			p.ByName[le.Name] = leaf
		}
	}
	for _, l := range p.Leaves {
		if !listed[l.Dir] {
			p.Diags.Errorf(diag.Pos{File: l.Dir + "/component.vulcan", Line: 1, Col: 1}, "K0003", "leaf directory %s is not listed in any container of system.vulcan", l.Dir)
		}
	}
	for _, l := range p.Leaves {
		if l.Comp == nil {
			continue
		}
		seen := map[string]int{}
		for _, g := range l.Comp.Gateways {
			if nodes[g.Name] != "external" {
				p.Diags.Errorf(l.CompSrc.Pos(g.Pos), "K0206", "gateway for %s: %s is not an external declared in system.vulcan", g.Name, g.Name)
			}
			seen[g.Name]++
		}
		for _, e := range l.Comp.Entries {
			if nodes[e.Name] == "" {
				p.Diags.Errorf(l.CompSrc.Pos(e.Pos), "K0206", "entry for %s: %s is not a person or an external declared in system.vulcan", e.Name, e.Name)
			}
		}
	}
	// One gateway for each external in the container.
	if p.Container != nil {
		gw := map[string]*Package{}
		for _, le := range p.Container.Leaves {
			l := p.ByName[le.Name]
			if l == nil || l.Comp == nil {
				continue
			}
			for _, g := range l.Comp.Gateways {
				if other, ok := gw[g.Name]; ok && other != l {
					p.Diags.Errorf(l.CompSrc.Pos(g.Pos), "K0207", "two gateway leaves for %s in container %s: %s and %s", g.Name, p.Container.Name, other.Name, l.Name)
				}
				gw[g.Name] = l
			}
		}
		for _, w := range p.Container.Wires {
			for _, e := range []decl.Endpoint{w.Need, w.Supply} {
				if e.Container != "" {
					p.Deferred(pos(e.Pos), "an endpoint with a container", "§5.5")
				}
				if groups[e.Leaf] {
					p.Diags.Errorf(pos(e.Pos), "K0113", "the wire names group %s; a group is only a label, so wire a leaf", e.Leaf)
				}
			}
			if w.Via != "" {
				p.Deferred(pos(w.ViaPos), "via on a wire", "§5.5.1")
			}
		}
	}
}

var configKeys = map[string]string{
	"context_budget":       "ok",
	"shutdown_timeout":     "ok",
	"on_contract_failure":  "ok",
	"timeout":              "§5.5.3",
	"transport":            "§5.5.1",
	"output":               "§11",
	"network":              "§5.4c",
	"acquiring":            "§5.4c",
	"allow_network":        "§5.4c",
	"allow_acquiring":      "§5.4c",
	"strict":               "§5.4d",
	"strict_exempt":        "§5.4d",
	"strict_interfaces":    "§5.4d",
	"strict_shared":        "§5.4d",
	"mutators":             "§5.4d",
	"duplicate_similarity": "§9",
	"duplicate_min_tokens": "§9",
}

func (p *Project) checkConfig(s *decl.Settings) {
	for _, it := range s.Items {
		sec, ok := configKeys[it.Key]
		switch {
		case !ok:
			p.Diags.Errorf(p.SysSrc.Pos(it.KeyPos), "K0006", "unknown config key %s", it.Key)
		case sec != "ok":
			p.Deferred(p.SysSrc.Pos(it.KeyPos), "config key "+it.Key, sec)
		}
	}
}

// Config returns the value of a config key, or nil.
func (p *Project) Config(key string) *decl.Value {
	if p.System == nil || p.System.Config == nil {
		return nil
	}
	for _, it := range p.System.Config.Items {
		if it.Key == key {
			return it.Value
		}
	}
	return nil
}
