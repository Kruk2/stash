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

import (
	"path/filepath"
	"strings"
)

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

// Unmap is the inverse of Map: it rewrites a "/mnt/<letter>" Linux path back to
// its Windows drive form ("M:\\..."). This is used when writing paths to, or
// querying, the database, which stores the original Windows-style paths so the
// data created on Windows is served unmodified.
//
// Only paths under "/mnt/<single letter>/" are translated; anything else (e.g.
// "/mnt/data/...", or a native "/srv/media") is returned unchanged so a
// Linux-native library keeps working too. The drive letter is upper-cased to
// match how Windows stores it.
func Unmap(p string) string {
	const prefix = "/mnt/"
	if !strings.HasPrefix(p, prefix) {
		return p
	}
	rest := p[len(prefix):]
	if rest == "" {
		return p
	}
	c := rest[0]
	if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
		return p
	}
	// must be a single-letter mount: the letter is followed by '/' or end of string
	if len(rest) > 1 && rest[1] != '/' {
		return p
	}
	drive := c &^ 0x20 // ASCII upper-case
	tail := strings.ReplaceAll(rest[1:], "/", "\\")
	return string(drive) + ":" + tail
}

// Sep returns the path separator appropriate for p. Windows drive paths produced
// by Unmap ("M:\\...") use a backslash; everything else uses the OS separator.
// Used when building LIKE patterns against the Windows-style paths in the DB.
func Sep(p string) string {
	if len(p) >= 2 && p[1] == ':' {
		return "\\"
	}
	return string(filepath.Separator)
}
