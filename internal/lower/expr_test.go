package lower

import "testing"

func TestParseContract(t *testing.T) {
	cases := map[string]string{
		"amt > 0 && amt < money.MaxOrder":           "amt > 0 && amt < money.MaxOrder",
		"err == nil => result.Amount == amount":     "(!(err == nil) || (result.Amount == amount))",
		"a => b => c":                               "(!(a) || (!(b) || (c)))",
		"x && (b => c)":                             "x && (!(b) || (c))",
		"err != nil => a.Balance == old(a.Balance)": "(!(err != nil) || (a.Balance == old(a.Balance)))",
		"len(result.0) == old(len(xs)) + 1":         "len(vulcan_result_0) == old(len(xs))+1",
		"f(a => b, c)":                              "f((!(a) || (b)), c)",
		`s != "=>"`:                                 `s != "=>"`,
	}
	for in, want := range cases {
		e, err := ParseContract(in)
		if err != nil {
			t.Errorf("%s: %v", in, err)
			continue
		}
		if got := GoText(e); got != want {
			t.Errorf("%s:\n got %s\nwant %s", in, got, want)
		}
	}
}
