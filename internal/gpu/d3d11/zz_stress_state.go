//go:build windows

package d3d11

import "fmt"

// StressState describes the pool and what New took from it (DRAFT).
func StressState() string {
	pool.Lock()
	k := pool.kit
	pool.Unlock()
	kitState := "empty"
	if k != nil {
		kitState = fmt.Sprintf("kit warp=%v luid=%#x", k.warp, k.luid)
	}
	return fmt.Sprintf("pool %s; devices made %d, shader sets %d, GPU tried with a WARP kit %d, WARP kit taken %d, GPU kit taken %d, GPU kit dropped %d",
		kitState, devicesMade.Load(), shadersMade.Load(), stressTried.Load(), stressWARPTaken.Load(), stressGPUTaken.Load(), stressGPUDropped.Load())
}
