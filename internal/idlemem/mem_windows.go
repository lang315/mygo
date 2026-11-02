package main

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

type state struct{}

func before() state { return state{} }

var getProcessMemoryInfo = syscall.NewLazyDLL("kernel32.dll").NewProc("K32GetProcessMemoryInfo")

// processMemoryCountersEx2 is PROCESS_MEMORY_COUNTERS_EX2, of Windows 10
// 1809 and later.
type processMemoryCountersEx2 struct {
	cb                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
	PrivateUsage               uintptr
	PrivateWorkingSetSize      uintptr
	SharedCommitUsage          uint64
}

const (
	processTerminate               = 0x0001
	processQueryLimitedInformation = 0x1000
)

// usage counts the app's descendants: WebView2's browser process, and the
// renderer, GPU and utility processes it runs.
func usage(pid int, _ state) (app, total uint64, procs []int, err error) {
	procs = append([]int{pid}, descendants(pid)...)
	for i, p := range procs {
		ws, err := proportionalWorkingSet(p)
		if err != nil {
			if i == 0 {
				return 0, 0, nil, err
			}
			continue
		}
		if i == 0 {
			app = ws
		}
		total += ws
	}
	return app, total, procs, nil
}

// proportionalWorkingSet returns the pages of pid in memory, each divided by
// the processes that share it (the working set's share count, which Windows
// caps at 7): the number Linux's proportional set size is, so that the DLLs and
// sections WebView2's processes share count once between them. It falls back
// to the private working set for a process it may not read.
func proportionalWorkingSet(pid int) (uint64, error) {
	const (
		queryInformation = 0x0400
		vmRead           = 0x0010
		errBadLength     = syscall.Errno(24)
		pageSize         = 4096
	)
	h, err := syscall.OpenProcess(queryInformation|vmRead, false, uint32(pid))
	if err != nil {
		return privateWorkingSet(pid)
	}
	defer syscall.CloseHandle(h)
	// PSAPI_WORKING_SET_INFORMATION: the number of entries, then each one's
	// flags: protection in bits 0-4, share count in bits 5-7, shared in bit 8.
	buf := make([]uintptr, 1<<16)
	for {
		ok, _, err := queryWorkingSet.Call(uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf))*unsafe.Sizeof(buf[0]))
		if ok != 0 {
			break
		}
		if err != errBadLength {
			return privateWorkingSet(pid)
		}
		buf = make([]uintptr, int(buf[0])*5/4+1024)
	}
	var total float64
	for _, e := range buf[1 : 1+buf[0]] {
		share := e >> 5 & 7
		if e&(1<<8) == 0 || share == 0 {
			share = 1
		}
		total += pageSize / float64(share)
	}
	return uint64(total), nil
}

var queryWorkingSet = syscall.NewLazyDLL("kernel32.dll").NewProc("K32QueryWorkingSet")

// privateWorkingSet returns the pages of pid in memory that no other
// process shares.
func privateWorkingSet(pid int) (uint64, error) {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return 0, err
	}
	defer syscall.CloseHandle(h)
	var c processMemoryCountersEx2
	c.cb = uint32(unsafe.Sizeof(c))
	if ok, _, err := getProcessMemoryInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&c)), uintptr(c.cb)); ok == 0 {
		return 0, err
	}
	return uint64(c.PrivateWorkingSetSize), nil
}

// descendants returns the processes pid started, and theirs.
func descendants(pid int) []int {
	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer syscall.CloseHandle(snap)
	children := map[int][]int{}
	var e syscall.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	for err := syscall.Process32First(snap, &e); err == nil; err = syscall.Process32Next(snap, &e) {
		children[int(e.ParentProcessID)] = append(children[int(e.ParentProcessID)], int(e.ProcessID))
	}
	// A parent's ID may have been reused by a process it did not start.
	seen := map[int]bool{pid: true}
	var out []int
	for queue := children[pid]; len(queue) > 0; queue = queue[1:] {
		if seen[queue[0]] {
			continue
		}
		seen[queue[0]] = true
		out = append(out, queue[0])
		queue = append(queue, children[queue[0]]...)
	}
	return out
}

// stop quits the app and the processes it ran, and returns once all have
// exited, with those they started since usage looked. It asks first, as
// closing the app's windows does, which Windows has for a signal: the app
// quits, and WebView2's processes with it. Killing them all, every few
// seconds, hung Windows CI runners now and then. What is left after a few
// seconds, or if the app shows no window, is ended with TerminateProcess,
// which only starts ending a process, and the next run's WebView2 could
// otherwise meet this run's browser still exiting, which holds the user
// data folder they share.
func stop(_ *os.Process, procs []int) error {
	app := procs[0]
	if closeWindows(app) {
		if h, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(app)); err == nil {
			syscall.WaitForSingleObject(h, quitWait)
			syscall.CloseHandle(h)
		}
	}
	// A handle on a process keeps its ID from going to another process,
	// which descendants would take for one of the app's.
	held := map[int]syscall.Handle{}
	defer func() {
		for _, h := range held {
			syscall.CloseHandle(h)
		}
	}()
	for range 10 {
		var ending []syscall.Handle
		for _, pid := range procs {
			if _, ok := held[pid]; ok {
				continue
			}
			h, err := syscall.OpenProcess(processTerminate|syscall.SYNCHRONIZE, false, uint32(pid))
			if err != nil {
				continue // gone already
			}
			held[pid] = h
			ending = append(ending, h)
		}
		if len(ending) == 0 {
			return nil
		}
		for _, h := range ending {
			syscall.TerminateProcess(h, 1)
		}
		for _, h := range ending {
			if ev, _ := syscall.WaitForSingleObject(h, 10_000); ev != syscall.WAIT_OBJECT_0 {
				return errors.New("a process of the app still runs 10s after it was ended")
			}
		}
		if _, ok := held[app]; !ok {
			return nil // its ID may be another process's now
		}
		procs = descendants(app)
	}
	return errors.New("the app's processes kept starting others as they were ended")
}

// quitWait is how long an app has to quit, in milliseconds, once its windows
// are closed.
const quitWait = 5_000

var (
	user32                   = syscall.NewLazyDLL("user32.dll")
	enumWindows              = user32.NewProc("EnumWindows")
	getWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	isWindowVisible          = user32.NewProc("IsWindowVisible")
	postMessage              = user32.NewProc("PostMessageW")
)

const wmClose = 0x0010

// closing is the process whose windows closeEach closes, and how many it
// did.
var closing struct{ pid, n int }

// closeEach is the one callback EnumWindows calls, as Windows keeps a
// callback made for as long as the process runs.
var closeEach = syscall.NewCallback(func(h, _ uintptr) uintptr {
	var pid uint32
	getWindowThreadProcessID.Call(h, uintptr(unsafe.Pointer(&pid)))
	if int(pid) == closing.pid {
		if visible, _, _ := isWindowVisible.Call(h); visible != 0 {
			postMessage.Call(h, wmClose, 0, 0)
			closing.n++
		}
	}
	return 1 // go on
})

// closeWindows closes the windows pid shows, and reports whether it shows
// one.
func closeWindows(pid int) bool {
	closing.pid, closing.n = pid, 0
	enumWindows.Call(closeEach, 0)
	return closing.n > 0
}

