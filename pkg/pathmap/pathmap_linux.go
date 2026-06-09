//go:build linux

// Package pathmap translates Windows-style absolute paths that are stored in
// the database (e.g. "F:\Movies\a.mp4") into native Linux paths
// ("/mnt/f/Movies/a.mp4") at read time. This lets a Stash database that was
// created on Windows be served unmodified on Linux.
//
// The rule is purely derived from the drive letter: "X:" maps to
// "/mnt/<lowercase x>/", and backslashes are converted to forward slashes.
// Paths that do not start with a "X:" drive prefix (i.e. already-native Linux
// paths) are returned unchanged, so a Linux-native library keeps working too.
package pathmap

import "strings"

// Map rewrites a Windows drive path to its /mnt/<letter> equivalent.
func Map(p string) string {
	if len(p) < 2 || p[1] != ':' {
		return p
	}
	c := p[0]
	if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')) {
		return p
	}
	letter := c | 0x20 // ASCII lower-case
	rest := strings.ReplaceAll(p[2:], "\\", "/")
	if rest == "" || rest[0] != '/' {
		rest = "/" + rest
	}
	return "/mnt/" + string(letter) + rest
}
