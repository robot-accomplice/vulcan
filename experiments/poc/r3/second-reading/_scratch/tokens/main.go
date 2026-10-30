// Counts Go-scanner tokens per file set, excluding comments. Works on .go and .vulcan.
package main

import (
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func count(path string) int {
	b, _ := os.ReadFile(path)
	fs := token.NewFileSet()
	f := fs.AddFile(path, -1, len(b))
	var s scanner.Scanner
	s.Init(f, b, func(token.Position, string) {}, 0)
	n := 0
	for {
		_, t, lit := s.Scan()
		if t == token.EOF {
			return n
		}
		if t == token.SEMICOLON && lit == "\n" {
			continue
		}
		n++
	}
}

func main() {
	root := os.Args[1]
	groups := map[string]int{}
	total := 0
	filepath.Walk(root, func(p string, i os.FileInfo, e error) error {
		if e != nil || i.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, ".vulcan") {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		n := count(p)
		groups[filepath.ToSlash(rel)] = n
		total += n
		return nil
	})
	for k, v := range groups {
		fmt.Printf("%6d %s\n", v, k)
	}
	fmt.Printf("%6d TOTAL\n", total)
}
