//go:build linux

package pathmap

import "testing"

func TestMap(t *testing.T) {
	cases := map[string]string{
		`F:\Ahegao\a.mp4`:  "/mnt/f/Ahegao/a.mp4", // backslash windows path
		`M:\x`:             "/mnt/m/x",
		`N:\`:              "/mnt/n/",
		`F:\Ahegao/a.mp4`:  "/mnt/f/Ahegao/a.mp4", // mixed seps (post filepath.Join)
		`c:\Users\domin`:   "/mnt/c/Users/domin", // lower-case drive
		"/mnt/f/native":    "/mnt/f/native",      // already linux -> unchanged
		"relative/path.mp4": "relative/path.mp4", // no drive -> unchanged
		"":                 "",
	}
	for in, want := range cases {
		if got := Map(in); got != want {
			t.Errorf("Map(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUnmap(t *testing.T) {
	cases := map[string]string{
		"/mnt/f/Ahegao/a.mp4": `F:\Ahegao\a.mp4`, // round-trips Map
		"/mnt/m/x":            `M:\x`,
		"/mnt/n/":             `N:\`,
		"/mnt/data/x":         "/mnt/data/x",    // multi-char mount -> unchanged
		"/srv/media":          "/srv/media",     // native linux -> unchanged
		"relative/path.mp4":   "relative/path.mp4",
		"":                    "",
	}
	for in, want := range cases {
		if got := Unmap(in); got != want {
			t.Errorf("Unmap(%q) = %q, want %q", in, got, want)
		}
	}

	// Map and Unmap should round-trip for drive paths.
	for _, p := range []string{`F:\Ahegao\a.mp4`, `M:\x`, `N:\`} {
		if got := Unmap(Map(p)); got != p {
			t.Errorf("Unmap(Map(%q)) = %q, want %q", p, got, p)
		}
	}
}

func TestSep(t *testing.T) {
	if got := Sep(`M:\x`); got != "\\" {
		t.Errorf("Sep(windows) = %q, want backslash", got)
	}
	if got := Sep("/mnt/data"); got != "/" {
		t.Errorf("Sep(linux) = %q, want slash", got)
	}
}
