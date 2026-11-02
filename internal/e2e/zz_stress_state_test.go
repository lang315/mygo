//go:build windows && (amd64 || arm64)

package e2e

func stressState() string { return "main: no device pool" }
