// Package main is a variant of the start-to-main measurement (TEMPORARY).
package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	if p := os.Getenv("MARK"); p != "" {
		if f, err := os.Create(p); err == nil {
			fmt.Fprintf(f, "%d", time.Now().UnixNano())
			f.Close()
		}
	}
}
