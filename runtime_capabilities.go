//go:build !server

package main

// isServerMode reports whether this binary was built for headless server mode (-tags server).
func isServerMode() bool {
	return false
}
