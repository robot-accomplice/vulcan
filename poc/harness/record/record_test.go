package record

import "testing"

// TestModelError is audit N33: a code of the shared model check in the final check is no success in
// either arm, and a code that one arm alone emits is counted apart.
func TestModelError(t *testing.T) {
	pass := Tests{Passed: 2}
	runs := []Run{
		{Arm: ArmL, Builds: true, Acceptance: pass},
		{Arm: ArmL, Builds: true, Acceptance: pass, FinalCheck: &FinalCheck{Fired: []string{"ARCH-07", "model-0101"}, Model: []string{"model-0101"}}},
		{Arm: ArmL, Builds: true, Acceptance: pass, FinalCheck: &FinalCheck{Fired: []string{"model-0907"}, OneArm: []string{"model-0907"}}},
		{Arm: ArmV, Builds: false, Acceptance: Tests{}, FinalCheck: &FinalCheck{Fired: []string{"model-0101"}, Model: []string{"model-0101"}}},
		{Arm: ArmV, Builds: true, Acceptance: pass, FinalCheck: &FinalCheck{Passed: true}},
	}
	for i, want := range []bool{true, false, true, false, true} {
		if got := runs[i].TaskSuccess(); got != want {
			t.Errorf("run %d: success %v, want %v", i, got, want)
		}
	}
	c := CountModel(runs)
	if c.Decided[ArmL] != 1 || c.Decided[ArmV] != 0 || c.OneArm[ArmL] != 1 || c.OneArm[ArmV] != 0 {
		t.Errorf("counts %+v", c)
	}
	// N42: Model counts a shared model code in each arm, so V's run counts too.
	if c.Model[ArmL] != 1 || c.Model[ArmV] != 1 {
		t.Errorf("model counts %+v", c.Model)
	}
}
