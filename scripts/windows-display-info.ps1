# Print what the Windows host looks like to Lumen: OS, GPU, monitors with DPI, WebView2, Office, NDI.
# Run from WSL:  powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$(wslpath -w scripts/windows-display-info.ps1)"
# Read-only: it writes nothing. Used to record the host in docs/windows-dev.md.
$ErrorActionPreference = "SilentlyContinue"
Add-Type -AssemblyName System.Windows.Forms
Add-Type @"
using System; using System.Runtime.InteropServices; using System.Collections.Generic;
public class LumenDpi {
  public delegate bool Cb(IntPtr m, IntPtr h, IntPtr r, IntPtr d);
  [DllImport("user32.dll")] public static extern bool EnumDisplayMonitors(IntPtr h, IntPtr r, Cb cb, IntPtr d);
  [DllImport("shcore.dll")] public static extern int GetScaleFactorForMonitor(IntPtr m, out int f);
  [DllImport("shcore.dll")] public static extern int GetDpiForMonitor(IntPtr m, int t, out uint x, out uint y);
  [DllImport("user32.dll")] public static extern IntPtr SetThreadDpiAwarenessContext(IntPtr c);
  public static List<string> Run() {
    var l = new List<string>();
    SetThreadDpiAwarenessContext((IntPtr)(-4)); // per-monitor v2
    EnumDisplayMonitors(IntPtr.Zero, IntPtr.Zero, (m,h,r,d) => {
      int f; GetScaleFactorForMonitor(m, out f);
      uint x, y; GetDpiForMonitor(m, 0, out x, out y);
      uint rx, ry; GetDpiForMonitor(m, 2, out rx, out ry);
      l.Add("  monitor " + m + ": GetScaleFactorForMonitor=" + f + "%  effectiveDPI=" + x + "  rawDPI=" + rx);
      return true; }, IntPtr.Zero);
    return l;
  }
}
"@
$os = Get-CimInstance Win32_OperatingSystem
"OS: $($os.Caption) build $($os.BuildNumber)"
"GPU: " + ((Get-CimInstance Win32_VideoController | % { $_.Name }) -join "; ")
"Screens (WinForms, physical px after the per-monitor call below):"
[System.Windows.Forms.Screen]::AllScreens | % { "  $($_.DeviceName) $($_.Bounds.Width)x$($_.Bounds.Height) at ($($_.Bounds.X),$($_.Bounds.Y)) primary=$($_.Primary)" }
"Per-monitor DPI:"
[LumenDpi]::Run()
"WebView2 runtime: " + (Get-ItemProperty "HKLM:\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}").pv
"PowerPoint ProgID: " + (Get-ItemProperty "Registry::HKEY_CLASSES_ROOT\PowerPoint.Application\CurVer")."(default)"
$c2r = Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Office\ClickToRun\Configuration"
"Office: $($c2r.ProductReleaseIds) $($c2r.VersionToReport)"
"NDI runtime env: V6=$env:NDI_RUNTIME_DIR_V6"
foreach ($t in "go","git","gh","gst-launch-1.0","ffmpeg","wails3") { $p = (Get-Command "$t.exe").Source; "tool $t : " + $(if ($p) { $p } else { "-" }) }
"Go: " + (& "C:\Program Files\Go\bin\go.exe" version)
