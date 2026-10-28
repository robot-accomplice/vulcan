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

// Make returns the slots of round 1: arms V and G.
func Make(tasks []string, runs int, seed uint64) []Slot {
	return MakeArms(tasks, []string{record.ArmV, record.ArmG}, runs, seed)
}

// MakeArms returns runs × arms slots for each task, shuffled by seed. The same seed gives the
// same order on every machine.
func MakeArms(tasks, arms []string, runs int, seed uint64) []Slot {
	var out []Slot
	for _, t := range tasks {
		for _, arm := range arms {
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

// RetestOffset marks a test-retest slot: its index is the original's plus RetestOffset.
const RetestOffset = 100

// AddRetest adds a second run of a share of the slots (round 2, test-retest: plan-r2 section 5),
// chosen by seed, and shuffles them in. The analysis keeps them out of the main measures and
// reports how often the two runs of a slot agree.
func AddRetest(slots []Slot, share float64, seed uint64) []Slot {
	r := rand.New(rand.NewPCG(seed^0x2545f4914f6cdd1d, seed))
	pick := r.Perm(len(slots))[:int(float64(len(slots))*share+0.5)]
	out := append([]Slot{}, slots...)
	for _, i := range pick {
		s := slots[i]
		s.Index += RetestOffset
		out = append(out, s)
	}
	r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	for i := range out {
		out[i].Order = i + 1
	}
	return out
}

// Shuffle orders slots by seed and numbers them again.
func Shuffle(slots []Slot, seed uint64) []Slot {
	out := append([]Slot{}, slots...)
	r := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	for i := range out {
		out[i].Order = i + 1
	}
	return out
}
