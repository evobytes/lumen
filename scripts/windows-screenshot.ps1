# Capture the whole virtual desktop (all monitors, physical pixels) to a PNG.
# Run from WSL (the output path can be inside the repo, as a \\wsl.localhost\... path):
#   scripts/windows-screenshot dist/selftest/desktop.png
param([Parameter(Mandatory = $true)][string]$Out)
Add-Type -AssemblyName System.Windows.Forms, System.Drawing
Add-Type @"
using System; using System.Runtime.InteropServices;
public class LumenShot { [DllImport("user32.dll")] public static extern bool SetProcessDPIAware(); }
"@
[void][LumenShot]::SetProcessDPIAware()
$b = [System.Windows.Forms.SystemInformation]::VirtualScreen
$bmp = New-Object System.Drawing.Bitmap $b.Width, $b.Height
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.CopyFromScreen($b.Left, $b.Top, 0, 0, $bmp.Size)
$bmp.Save($Out, [System.Drawing.Imaging.ImageFormat]::Png)
"virtual screen: $($b.Left),$($b.Top) $($b.Width)x$($b.Height) -> $Out"
