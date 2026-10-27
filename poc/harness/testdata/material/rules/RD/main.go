// Command rd detects a database opened outside the store part.
package main

import "github.com/robot-accomplice/vulcan/poc/harness/detect"

func main() {
	detect.Main(func(s *detect.Source) []detect.Violation {
		var out []detect.Violation
		for _, u := range s.UsesOf("database/sql.Open") {
			if u.Unit != "api/store" {
				out = append(out, detect.Violation{Pos: u.Pos, Text: "a database is opened outside store"})
			}
		}
		return out
	})
}
