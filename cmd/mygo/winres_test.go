package main

import "testing"

func TestWindowsVersion(t *testing.T) {
	for v, want := range map[string][4]uint16{
		"1.2.3":          {1, 2, 3, 0},
		"v1.2.3":         {1, 2, 3, 0},
		"1.2.3+5":        {1, 2, 3, 0},
		"1.2.3+build-5":  {1, 2, 3, 0},
		"1.70000.0":      {1, 65535, 0, 0},
		"1.2.3.4":        {1, 2, 3, 4},
		"99999999999.1":  {65535, 1, 0, 0},
		"1.2.3-beta+exp": {1, 2, 3, 0},
		// A prerelease is not newer than its release to Windows installers.
		"1.2.3-beta.4": {1, 2, 3, 0},
	} {
		if got := windowsVersion(v); got != want {
			t.Errorf("windowsVersion(%q) = %v, want %v", v, got, want)
		}
	}
}
