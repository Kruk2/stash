//go:build !linux

package pathmap

import "path/filepath"

// Map is a no-op on non-Linux platforms: paths are used exactly as stored in
// the database. This keeps the Windows build byte-for-byte unaffected.
func Map(p string) string { return p }

// Unmap is a no-op on non-Linux platforms.
func Unmap(p string) string { return p }

// Sep returns the native OS path separator on non-Linux platforms.
func Sep(p string) string { return string(filepath.Separator) }
