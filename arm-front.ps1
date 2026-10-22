Add-Type @"
using System;
using System.Text;
using System.Runtime.InteropServices;
public static class Front {
  public delegate bool EnumProc(IntPtr h, IntPtr l);
  [DllImport("user32.dll")] static extern IntPtr GetForegroundWindow();
  [DllImport("user32.dll")] static extern bool EnumWindows(EnumProc p, IntPtr l);
  [DllImport("user32.dll")] static extern bool IsWindowVisible(IntPtr h);
  [DllImport("user32.dll")] static extern IntPtr WindowFromPoint(POINT p);
  [DllImport("user32.dll")] static extern bool SetCursorPos(int x, int y);
  [DllImport("user32.dll")] static extern void mouse_event(uint f, uint x, uint y, uint d, UIntPtr e);
  [DllImport("user32.dll")] static extern void keybd_event(byte vk, byte scan, uint flags, UIntPtr e);
  [DllImport("user32.dll", CharSet=CharSet.Unicode)] static extern int GetClassName(IntPtr h, StringBuilder s, int n);
  [DllImport("user32.dll", CharSet=CharSet.Unicode)] static extern int GetWindowText(IntPtr h, StringBuilder s, int n);
  [DllImport("user32.dll")] static extern bool PostMessage(IntPtr h, uint m, IntPtr w, IntPtr l);
  [DllImport("user32.dll")] static extern bool GetWindowRect(IntPtr h, out RECT r);
  [DllImport("dwmapi.dll")] static extern int DwmGetWindowAttribute(IntPtr h, int a, out int v, int n);
  [StructLayout(LayoutKind.Sequential)] public struct POINT { public int X, Y; }
  [StructLayout(LayoutKind.Sequential)] public struct RECT { public int L, T, R, B; }
  public static string Describe(IntPtr h) {
    var c = new StringBuilder(256); var t = new StringBuilder(256);
    GetClassName(h, c, 256); GetWindowText(h, t, 256);
    RECT r; GetWindowRect(h, out r);
    int cloaked = 0; DwmGetWindowAttribute(h, 14, out cloaked, 4);
    return string.Format("{0} '{1}' visible={2} cloaked={3} rect=({4},{5},{6},{7})", c, t, IsWindowVisible(h), cloaked, r.L, r.T, r.R, r.B);
  }
  public static string Now() { return Describe(GetForegroundWindow()); }
  public static bool IsCore() { return Now().StartsWith("Windows.UI.Core.CoreWindow"); }
  public static string At(int x, int y) { POINT p; p.X = x; p.Y = y; return Describe(WindowFromPoint(p)); }
  public static string All() {
    var sb = new StringBuilder();
    EnumWindows((h, l) => { if (IsWindowVisible(h)) sb.AppendLine("  " + Describe(h)); return true; }, IntPtr.Zero);
    return sb.ToString();
  }
  public static bool CloseFirstRun() {
    var h = GetForegroundWindow();
    var c = new StringBuilder(256); var t = new StringBuilder(256);
    GetClassName(h, c, 256); GetWindowText(h, t, 256);
    if (c.ToString() != "Windows.UI.Core.CoreWindow" || t.ToString() != "Microsoft account") return false;
    return PostMessage(h, 0x10, IntPtr.Zero, IntPtr.Zero);
  }
  public static void Key(byte vk) { keybd_event(vk, 0, 0, UIntPtr.Zero); keybd_event(vk, 0, 2, UIntPtr.Zero); }
  public static void Click(int x, int y) { SetCursorPos(x, y); mouse_event(2, 0, 0, 0, UIntPtr.Zero); mouse_event(4, 0, 0, 0, UIntPtr.Zero); }
  public static void WinD() { keybd_event(0x5B, 0, 0, UIntPtr.Zero); Key(0x44); keybd_event(0x5B, 0, 2, UIntPtr.Zero); }
}
"@

"in front: " + [Front]::Now()
if ([Front]::CloseFirstRun()) {
  Start-Sleep -Seconds 2
  "closed it; in front: " + [Front]::Now()
}
"visible windows:"
[Front]::All()
"at (500,700): " + [Front]::At(500, 700) + "; at (60,230): " + [Front]::At(60, 230)

Start-Sleep -Seconds 5
"after 5 s; in front: " + [Front]::Now()

if ([Front]::IsCore()) {
  [Front]::Key(0x1B)
  Start-Sleep -Seconds 2
  "after Escape; in front: " + [Front]::Now()
}
if ([Front]::IsCore()) {
  [Front]::Click(500, 700)
  Start-Sleep -Seconds 2
  "after a click on the desktop; in front: " + [Front]::Now()
}
if ([Front]::IsCore()) {
  [Front]::WinD()
  Start-Sleep -Seconds 2
  "after Win+D; in front: " + [Front]::Now()
}
"visible windows at the end:"
[Front]::All()
