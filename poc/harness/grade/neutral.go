package grade

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// A judge of a match must not see the arm (plan-r3, section 5). So a model check reaches the judge as
// one sentence for each digit of its code, the same whichever arm fired, and under an id with no digit,
// so that a code that one arm alone has reads like any other.
type modelCheck struct {
	text string
	arm  string // "V" or "L" for a code that one arm alone has, else ""
}

var modelChecks = map[string]modelCheck{
	"0001": {"A declaration or a contract has a syntax error.", "V"},
	"0002": {"A file or a package is in a place that the project's layout does not allow.", ""},
	"0003": {"A part has no package, or a part has another kind than its place allows.", ""},
	"0004": {"A package name does not match its directory.", ""},
	"0005": {"A name is used that is reserved for generated code.", "V"},
	"0006": {"The configuration has an unknown key.", "V"},
	"0008": {"The code does not compile.", "V"},
	"0009": {"The code uses a construct that this release does not support.", "V"},
	"0101": {"A leaf imports another leaf.", ""},
	"0104": {"A port has no provider.", ""},
	"0105": {"A port has more than one provider.", "V"},
	"0107": {"A wiring names something that does not exist.", "V"},
	"0108": {"A provider does not have the methods of its port.", "V"},
	"0110": {"The declaration of a port has a fault.", ""},
	"0111": {"The wiring has a cycle.", "V"},
	"0112": {"A port signature uses a type that is private to one side.", ""},
	"0113": {"A wiring names a group, not a leaf.", "V"},
	"0116": {"Code changes a port after it is set, or holds a port in the wrong place.", ""},
	"0117": {"Code runs before the ports are set.", ""},
	"0118": {"Code imports something that only the tools may use.", ""},
	"0120": {"The contract of a port names a private name of its leaf.", ""},
	"0121": {"A name or a part is declared twice.", ""},
	"0122": {"A link directive reaches code that the rules cannot see.", ""},
	"0123": {"A port carries a function, a channel, or an interface.", ""},
	"0124": {"Code uses unsafe, syscall, cgo, or a reflective call.", ""},
	"0125": {"A leaf that serves also dials.", ""},
	"0126": {"Some code is not Go: a function with no body, an assembly file, or an object file.", ""},
	"0201": {"A library declares ports.", ""},
	"0202": {"A library variable is assigned after startup.", ""},
	"0203": {"A library chooses a file or a resource by name.", ""},
	"0204": {"A library imports a leaf.", ""},
	"0205": {"A part that is not a gateway or an entry reaches the network, a database, or a process.", ""},
	"0206": {"A gateway or an entry is for something that is not declared.", ""},
	"0207": {"Two gateways serve one external system.", ""},
	"0209": {"A dependency is missing from the files of the module.", "V"},
	"0210": {"A library variable holds a reference.", ""},
	"0211": {"Code changes state that the whole process shares.", ""},
	"0501": {"The constant arguments of a call break a precondition of the callee.", ""},
	"0502": {"A contract reads a result with no guard on the error.", "V"},
	"0503": {"A contract or an invariant is not pure.", ""},
	"0506": {"The zero value breaks an invariant.", ""},
	"0507": {"A field of a type with an invariant is written outside its methods.", ""},
	"0508": {"The address of a field of a type with an invariant escapes its methods.", ""},
	"0509": {"A saved value in a contract holds a reference.", ""},
	"0510": {"A contract names a result that does not exist.", "V"},
	"0511": {"A function with a contract uses a reserved name.", "V"},
	"0512": {"A contract uses a type assertion.", ""},
	"0513": {"An invariant is on a type where it cannot hold.", ""},
	"0514": {"A function with a contract has no body.", "V"},
	"0601": {"A startup function has the wrong signature.", "V"},
	"0602": {"Two settings conflict in the flags that they use.", "V"},
	"0603": {"A setting has no value.", ""},
	"0604": {"A constructor takes something other than ports and settings.", ""},
	"0605": {"A secret setting has a constant value.", ""},
	"0606": {"A setting is not a constant or a flag value.", ""},
	"0607": {"One constructor makes a second instance.", ""},
	"0705": {"The tests ran with no race detector.", "V"},
	"0706": {"A standard analysis of the code reports a fault.", "V"},
	"0801": {"The rules file has a fault.", ""},
	"0811": {"An entry of the baseline is stale.", ""},
	"0901": {"A file is left out of the build.", ""},
	"0902": {"The declaration of the architecture has a fault.", "L"},
	"0903": {"A statement of a contract is not among the first statements.", "L"},
	"0904": {"A method of a type with an invariant does not keep its check.", "L"},
	"0905": {"A value of a type with an invariant is built with no check.", "L"},
	"0906": {"The check of a port's contract has the wrong form, or is missing.", "L"},
	"0907": {"The start of the program calls something that it may not call.", "L"},
	"0908": {"The start of the program builds the value of a leaf itself.", "L"},
	"0909": {"The start of the program makes a wiring in a form that cannot be read.", "L"},
}

// modelText gives the sentence of a digit.
func modelText(digits string) string {
	if c, ok := modelChecks[digits]; ok {
		return c.text
	}
	return "A check of the model rejects the code."
}

// judgeID gives the id that a judge sees for a rule that fired: a rule's own id, or for a model
// check an id with no digit of its code.
func judgeID(rule string) string {
	if len(rule) == len("model-0000") && rule[:6] == "model-" {
		sum := sha256.Sum256([]byte(rule))
		return "check-" + hex.EncodeToString(sum[:4])
	}
	return rule
}

// fromJudgeID reverses judgeID.
func fromJudgeID(id string) string {
	if len(id) > 6 && id[:6] == "check-" {
		for n := range 10000 {
			if rule := fmt.Sprintf("model-%04d", n); judgeID(rule) == id {
				return rule
			}
		}
	}
	return id
}
