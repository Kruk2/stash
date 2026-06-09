//go:build !linux

package pathmap

// Map is a no-op on non-Linux platforms: paths are used exactly as stored in
// the database. This keeps the Windows build byte-for-byte unaffected.
func Map(p string) string { return p }
