module github.com/robot-accomplice/vulcan

// go 1.26 is the minimum language version, not a toolchain pin: decision 5 supports Go 1.26
// and newer, and builds use the installed toolchain (Go 1.27 on 2026-10-27).
go 1.26.0

require (
	golang.org/x/mod v0.42.0
	golang.org/x/tools v0.52.0
)

require golang.org/x/sync v0.24.0 // indirect
