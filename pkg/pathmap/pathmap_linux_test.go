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
