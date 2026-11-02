//go:build !windows

package conformance

// shortPath gives the path: only Windows has 8.3 forms.
func shortPath(p string) string { return p }
