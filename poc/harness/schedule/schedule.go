// Package schedule orders the runs of the evaluation: every task, every arm, and every run
// index, in a random interleaved order fixed by a committed seed (docs/poc/plan.md, section 7).
package schedule

import (
	"math/rand/v2"

	"github.com/robot-accomplice/vulcan/poc/harness/record"
)

// Slot is one run to make.
type Slot struct {
	Order int    `json:"order"`
	Task  string `json:"task"`
	Arm   string `json:"arm"`
	Index int    `json:"index"`
}

// Make returns runs × arms slots for each task, shuffled by seed. The same seed gives the same
// order on every machine.
func Make(tasks []string, runs int, seed uint64) []Slot {
	var out []Slot
	for _, t := range tasks {
		for _, arm := range []string{record.ArmV, record.ArmG} {
			for i := 1; i <= runs; i++ {
				out = append(out, Slot{Task: t, Arm: arm, Index: i})
			}
		}
	}
	r := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	for i := range out {
		out[i].Order = i + 1
	}
	return out
}
