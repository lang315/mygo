// Command cost runs each exe 40 times and prints the median of process start
// to main (TEMPORARY).
package main

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"
)

func main() {
	mark := os.TempDir() + `\mark.txt`
	for _, exe := range os.Args[1:] {
		var ds []time.Duration
		for i := 0; i < 41; i++ {
			os.Remove(mark)
			cmd := exec.Command(exe)
			cmd.Env = append(os.Environ(), "MARK="+mark)
			t0 := time.Now()
			if err := cmd.Run(); err != nil {
				fmt.Println(exe, err)
				break
			}
			b, _ := os.ReadFile(mark)
			ns, _ := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
			if i > 0 { // the first run is cold
				ds = append(ds, time.Unix(0, ns).Sub(t0))
			}
		}
		slices.Sort(ds)
		st, _ := os.Stat(exe)
		fmt.Printf("%-28s size %8d  start->main median %6.2f ms (min %6.2f max %6.2f)\n", exe[strings.LastIndex(exe, `\`)+1:], st.Size(), ms(ds[len(ds)/2]), ms(ds[0]), ms(ds[len(ds)-1]))
	}
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }
