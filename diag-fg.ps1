Add-Type @"
using System;
using System.Text;
using System.Runtime.InteropServices;
public static class W {
  public delegate bool EnumProc(IntPtr h, IntPtr l);
  [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
  [DllImport("user32.dll")] public static extern IntPtr GetShellWindow();
  [DllImport("user32.dll")] public static extern bool EnumWindows(EnumProc p, IntPtr l);
  [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr h);
  [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr h, int cmd);
  [DllImport("user32.dll")] public static extern bool PostMessage(IntPtr h, uint m, IntPtr w, IntPtr l);
  [DllImport("user32.dll", CharSet=CharSet.Unicode)] public static extern int GetWindowText(IntPtr h, StringBuilder s, int n);
  [DllImport("user32.dll", CharSet=CharSet.Unicode)] public static extern int GetClassName(IntPtr h, StringBuilder s, int n);
  [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
  [DllImport("dwmapi.dll")] public static extern int DwmGetWindowAttribute(IntPtr h, int a, out int v, int n);
  [StructLayout(LayoutKind.Sequential)] public struct RECT { public int L, T, R, B; }
  public static string Describe(IntPtr h) {
    var c = new StringBuilder(256); var t = new StringBuilder(256);
    GetClassName(h, c, 256); GetWindowText(h, t, 256);
    uint pid; GetWindowThreadProcessId(h, out pid);
    RECT r; GetWindowRect(h, out r);
    int cloaked = 0; DwmGetWindowAttribute(h, 14, out cloaked, 4);
    return string.Format("{0:x} class={1} title='{2}' pid={3} visible={4} cloaked={5} rect=({6},{7},{8},{9})",
      h.ToInt64(), c, t, pid, IsWindowVisible(h), cloaked, r.L, r.T, r.R, r.B);
  }
  public static string All() {
    var sb = new StringBuilder();
    EnumWindows((h, l) => { if (IsWindowVisible(h)) sb.AppendLine("  " + Describe(h)); return true; }, IntPtr.Zero);
    return sb.ToString();
  }
}
"@

function Show($what) {
  $fg = [W]::GetForegroundWindow()
  "[$what] foreground: " + [W]::Describe($fg)
  $pid1 = 0
  [void][W]::GetWindowThreadProcessId($fg, [ref]$pid1)
  $p = Get-Process -Id $pid1 -ErrorAction SilentlyContinue
  if ($p) { "   process: $($p.ProcessName) $($p.Path)" }
}

"session: $((Get-Process -Id $PID).SessionId); user: $env:USERNAME"
Show "start"
"visible top-level windows:"
[W]::All()
"explorer: " + ((Get-Process explorer -ErrorAction SilentlyContinue | ForEach-Object { "$($_.Id)" }) -join ",")

$fg = [W]::GetForegroundWindow()
$pid1 = 0
[void][W]::GetWindowThreadProcessId($fg, [ref]$pid1)

# Remedy 1: ask the window to close (WM_CLOSE)
[void][W]::PostMessage($fg, 0x10, [IntPtr]::Zero, [IntPtr]::Zero)
Start-Sleep -Seconds 2
Show "after WM_CLOSE"

# Remedy 2: hide it
[void][W]::ShowWindow([W]::GetForegroundWindow(), 0)
Start-Sleep -Seconds 1
Show "after SW_HIDE"

# Remedy 3: show the desktop
(New-Object -ComObject Shell.Application).ToggleDesktop()
Start-Sleep -Seconds 2
Show "after ToggleDesktop"

# Remedy 4: stop the process that owns it
$p = Get-Process -Id $pid1 -ErrorAction SilentlyContinue
if ($p -and $p.ProcessName -ne "explorer") {
  Stop-Process -Id $pid1 -Force -ErrorAction SilentlyContinue
  Start-Sleep -Seconds 3
  Show "after Stop-Process $($p.ProcessName)"
}
"visible top-level windows at the end:"
[W]::All()
