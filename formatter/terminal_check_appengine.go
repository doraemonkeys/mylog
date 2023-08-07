// +build appengine

package formatter

import (
	"io"
)

func checkIfTerminal(w io.Writer) bool {
	return true
}
