// +build js nacl plan9

package formatter

import (
	"io"
)

func checkIfTerminal(w io.Writer) bool {
	return false
}
