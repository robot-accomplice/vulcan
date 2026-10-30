package declassify

import "samples/lab"

// Reveal is the one audited place that removes the label.
func Reveal(e lab.Email) string { return string(e) }
