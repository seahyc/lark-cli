//go:build !unix

package config

// lockDir is a no-op where advisory file locks are unavailable; writes are still
// atomic and serialised within the process.
func lockDir(dir string) (func(), error) {
	return func() {}, nil
}
