//go:build wasi
// +build wasi

package formatter

func isTerminal(fd int) bool {
	return false
}
