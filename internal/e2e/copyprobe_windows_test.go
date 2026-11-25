//go:build windows && (amd64 || arm64)

package e2e

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// PROBE (win-copy-flake): who holds the clipboard as the web view's browser
// process writes a Copy. Chromium's write tries OpenClipboard a few times
// and then gives up without a word; a holder that opened it with no window
// reads as "open by none" in GetOpenClipboardWindow. The probe tries to open
// the clipboard itself, from a thread of its own, about every millisecond,
// and closes it at once: a try that fails means another thread has it.

var (
	probeUser32                   = syscall.NewLazyDLL("user32.dll")
	probeKernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procProbeOpenClipboard        = probeUser32.NewProc("OpenClipboard")
	procProbeCloseClipboard       = probeUser32.NewProc("CloseClipboard")
	procProbeOpenClipboardWindow  = probeUser32.NewProc("GetOpenClipboardWindow")
	procProbeClipboardOwner       = probeUser32.NewProc("GetClipboardOwner")
	procProbeClipboardViewer      = probeUser32.NewProc("GetClipboardViewer")
	procProbeClipboardSequence    = probeUser32.NewProc("GetClipboardSequenceNumber")
	procProbeWindowThreadProcess  = probeUser32.NewProc("GetWindowThreadProcessId")
	procProbeClassName            = probeUser32.NewProc("GetClassNameW")
	procProbeSystemMetrics        = probeUser32.NewProc("GetSystemMetrics")
	procProbeCreateWaitableTimer  = probeKernel32.NewProc("CreateWaitableTimerExW")
	procProbeSetWaitableTimer     = probeKernel32.NewProc("SetWaitableTimer")
	procProbeWaitForSingleObject  = probeKernel32.NewProc("WaitForSingleObject")
	procProbeQueryFullProcessName = probeKernel32.NewProc("QueryFullProcessImageNameW")
)

type probeStretch struct {
	from, to time.Time
	samples  int
	windows  map[uintptr]int // GetOpenClipboardWindow while busy
}

type probeOwner struct {
	at   time.Time
	hwnd uintptr
}

type probeProc struct {
	name    string
	parent  uint32
	created time.Time
	cpu     time.Duration
}

type copyProbe struct {
	start  time.Time
	sentAt atomic.Int64 // UnixNano of Copy, 0 before
	quit   atomic.Bool
	done   chan struct{}
	before map[uint32]probeProc

	// Written by the sampler before done closes.
	samples   int
	maxGap    time.Duration
	end       time.Time
	moved     time.Time
	stretches []*probeStretch
	owners    []probeOwner

	once   sync.Once
	report string
}

// startCopyProbe starts the sampler; sent marks Copy, stop ends it.
func startCopyProbe() *copyProbe {
	p := &copyProbe{done: make(chan struct{}), before: probeProcesses()}
	p.start = time.Now()
	ready := make(chan struct{})
	go p.sample(ready)
	<-ready
	return p
}

func (p *copyProbe) sent(at time.Time) { p.sentAt.Store(at.UnixNano()) }

func (p *copyProbe) sample(ready chan struct{}) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(p.done)
	// A high resolution timer paces the tries at 1 ms without raising the
	// system's timer resolution, which would change how long Chromium's
	// own retries sleep.
	timer, _, _ := procProbeCreateWaitableTimer.Call(0, 0, 2, 0x1F0003) // CREATE_WAITABLE_TIMER_HIGH_RESOLUTION, TIMER_ALL_ACCESS
	if timer == 0 {
		timer, _, _ = procProbeCreateWaitableTimer.Call(0, 0, 0, 0x1F0003)
	}
	defer syscall.CloseHandle(syscall.Handle(timer))
	seq0, _, _ := procProbeClipboardSequence.Call()
	close(ready)
	var cur *probeStretch
	var owner uintptr
	last := time.Now()
	for !p.quit.Load() {
		now := time.Now()
		if g := now.Sub(last); g > p.maxGap {
			p.maxGap = g
		}
		last = now
		p.samples++
		if o, _, _ := procProbeClipboardOwner.Call(); o != owner || len(p.owners) == 0 {
			owner = o
			p.owners = append(p.owners, probeOwner{now, o})
		}
		if r, _, _ := procProbeOpenClipboard.Call(0); r != 0 {
			procProbeCloseClipboard.Call()
			cur = nil
		} else {
			w, _, _ := procProbeOpenClipboardWindow.Call()
			if cur == nil {
				cur = &probeStretch{from: now, windows: map[uintptr]int{}}
				p.stretches = append(p.stretches, cur)
			}
			cur.to = now
			cur.samples++
			cur.windows[w]++
		}
		if s, _, _ := procProbeClipboardSequence.Call(); s != seq0 {
			p.moved = now
			break
		}
		if sent := p.sentAt.Load(); sent != 0 && now.UnixNano()-sent > int64(300*time.Millisecond) {
			break
		}
		due := int64(-10000) // 1 ms, relative, in 100 ns
		procProbeSetWaitableTimer.Call(timer, uintptr(unsafe.Pointer(&due)), 0, 0, 0, 0)
		procProbeWaitForSingleObject.Call(timer, 0xFFFFFFFF)
	}
	p.end = time.Now()
}

// facts stops the sampler and returns how long after Copy it tried the
// clipboard, the longest gap between its tries, and whether one found the
// clipboard held, for copydrop.Skip.
func (p *copyProbe) facts() (sampled, maxGap time.Duration, busy bool) {
	p.stop()
	sent := p.sentAt.Load()
	if sent == 0 {
		return 0, p.maxGap, len(p.stretches) > 0
	}
	return p.end.Sub(time.Unix(0, sent)), p.maxGap, len(p.stretches) > 0
}

// webView2 is the path of the WebView2 runtime, which names its version.
func (*copyProbe) webView2() string { return probeWebView2Path() }

// busy reports whether the write never came: then the stop's report names
// the processes around it. A pass's busy try is the browser's own write.
func (p *copyProbe) busy() bool {
	<-p.done
	return p.moved.IsZero()
}

// stop ends the sampler and describes what it saw, times relative to Copy.
func (p *copyProbe) stop() string {
	if p == nil {
		return ""
	}
	p.once.Do(func() {
		p.quit.Store(true)
		<-p.done
		sentNano := p.sentAt.Load()
		sent := time.Unix(0, sentNano)
		rel := func(t time.Time) string {
			if sentNano == 0 {
				return fmt.Sprintf("%+dms from start", t.Sub(p.start).Milliseconds())
			}
			return fmt.Sprintf("%+.1fms", float64(t.Sub(sent).Microseconds())/1000)
		}
		after := probeProcesses()
		var b strings.Builder
		fmt.Fprintf(&b, "PROBE sampler: %d tries from %s to %s, max gap %v", p.samples, rel(p.start), rel(p.end), p.maxGap.Round(time.Microsecond))
		if p.moved.IsZero() {
			b.WriteString("; the sequence did not move")
		} else {
			fmt.Fprintf(&b, "; the sequence moved at %s", rel(p.moved))
		}
		if len(p.stretches) == 0 {
			b.WriteString("; never busy")
		}
		for _, s := range p.stretches {
			fmt.Fprintf(&b, "; busy %s to %s (%d tries), open by", rel(s.from), rel(s.to), s.samples)
			for w, n := range s.windows {
				fmt.Fprintf(&b, " %s x%d", probeWindow(w, after), n)
			}
		}
		b.WriteString("; owner")
		for _, o := range p.owners {
			fmt.Fprintf(&b, " %s at %s", probeWindow(o.hwnd, after), rel(o.at))
		}
		if !p.busy() {
			p.report = b.String()
			return
		}
		// Who could hold it with no window: the session, the old viewer
		// chain, the processes started around it and those that ran most.
		remote, _, _ := procProbeSystemMetrics.Call(0x1000) // SM_REMOTESESSION
		viewer, _, _ := procProbeClipboardViewer.Call()
		fmt.Fprintf(&b, "; remote session %v, clipboard viewer %s", remote != 0, probeWindow(viewer, after))
		var started, ran []string
		type use struct {
			pid uint32
			cpu time.Duration
		}
		var uses []use
		for pid, a := range after {
			if !a.created.Before(p.start.Add(-2 * time.Second)) {
				started = append(started, fmt.Sprintf("%s %d (parent %d) at %s", a.name, pid, a.parent, rel(a.created)))
			}
			if d := a.cpu - p.before[pid].cpu; d > 0 {
				uses = append(uses, use{pid, d})
			}
		}
		slices.SortFunc(uses, func(x, y use) int { return int(y.cpu - x.cpu) })
		for _, u := range uses[:min(8, len(uses))] {
			ran = append(ran, fmt.Sprintf("%s %d %v", after[u.pid].name, u.pid, u.cpu.Round(time.Millisecond)))
		}
		slices.Sort(started)
		fmt.Fprintf(&b, "; started since 2s before: [%s]; most CPU from start to end: [%s]", strings.Join(started, ", "), strings.Join(ran, ", "))
		var present []string
		for pid, a := range after {
			switch strings.ToLower(a.name) {
			case "rdpclip.exe", "ctfmon.exe", "textinputhost.exe", "clipup.exe", "clipesuconsumer.exe":
				present = append(present, fmt.Sprintf("%s %d", a.name, pid))
			}
		}
		slices.Sort(present)
		fmt.Fprintf(&b, "; suspects running: [%s]; WebView2: %s", strings.Join(present, ", "), probeWebView2Path())
		p.report = b.String()
	})
	return p.report
}

// probeWindow names a window: its handle, class, process and exe.
func probeWindow(w uintptr, procs map[uint32]probeProc) string {
	if w == 0 {
		return "0x0"
	}
	var pid uint32
	procProbeWindowThreadProcess.Call(w, uintptr(unsafe.Pointer(&pid)))
	var class [128]uint16
	procProbeClassName.Call(w, uintptr(unsafe.Pointer(&class[0])), uintptr(len(class)))
	self := ""
	if int(pid) == os.Getpid() {
		self = " (this one)"
	}
	return fmt.Sprintf("%#x %q of %s %d%s", w, syscall.UTF16ToString(class[:]), procs[pid].name, pid, self)
}

// probeProcesses returns every process's name, parent, start and CPU time.
func probeProcesses() map[uint32]probeProc {
	out := map[uint32]probeProc{}
	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return out
	}
	defer syscall.CloseHandle(snap)
	var e syscall.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	for err := syscall.Process32First(snap, &e); err == nil; err = syscall.Process32Next(snap, &e) {
		p := probeProc{name: syscall.UTF16ToString(e.ExeFile[:]), parent: e.ParentProcessID}
		if h, err := syscall.OpenProcess(0x1000, false, e.ProcessID); err == nil { // PROCESS_QUERY_LIMITED_INFORMATION
			var c, x, k, u syscall.Filetime
			if syscall.GetProcessTimes(h, &c, &x, &k, &u) == nil {
				p.created = time.Unix(0, c.Nanoseconds())
				p.cpu = time.Duration((int64(k.HighDateTime)<<32|int64(k.LowDateTime))+(int64(u.HighDateTime)<<32|int64(u.LowDateTime))) * 100
			}
			syscall.CloseHandle(h)
		}
		out[e.ProcessID] = p
	}
	return out
}

// probeWebView2Path is the path of a WebView2 process of this one, which
// names the runtime's version.
func probeWebView2Path() string {
	for _, pid := range webViewProcesses() {
		h, err := syscall.OpenProcess(0x1000, false, uint32(pid))
		if err != nil {
			continue
		}
		var buf [512]uint16
		n := uint32(len(buf))
		r, _, _ := procProbeQueryFullProcessName.Call(uintptr(h), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
		syscall.CloseHandle(h)
		if r != 0 {
			return syscall.UTF16ToString(buf[:n])
		}
	}
	return "unknown"
}
