; Inno Setup script for Lumen. Built by scripts/build-windows-installer, which passes:
;   AppVersion Arch Copyright Publisher SourceExe RepoRoot OutputDir OutputBase
; The binary is self-contained; the installer only copies files and creates shortcuts.
; Lumen is per-user configured (%APPDATA%\Lumen), so nothing is created outside Program Files.

#ifndef AppVersion
  #error AppVersion must be passed with /DAppVersion=...
#endif

[Setup]
; AppId identifies the product across upgrades: never change it.
AppId={{6C4F2B7E-3A1D-4E58-9B0C-1F7D2A9E5C44}
AppName=Lumen
AppVersion={#AppVersion}
AppPublisher={#Publisher}
AppCopyright={#Copyright}
DefaultDirName={autopf}\Lumen
DefaultGroupName=Lumen
UninstallDisplayIcon={app}\lumen.exe
OutputDir={#OutputDir}
OutputBaseFilename={#OutputBase}
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=admin
#if Arch == "arm64"
ArchitecturesAllowed=arm64
ArchitecturesInstallIn64BitMode=arm64
#else
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
#endif
LicenseFile={#RepoRoot}\LICENSE

[Files]
Source: "{#SourceExe}"; DestDir: "{app}"; DestName: "lumen.exe"; Flags: ignoreversion
Source: "{#RepoRoot}\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#RepoRoot}\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#RepoRoot}\NOTICE"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#RepoRoot}\THIRD_PARTY_NOTICES.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#RepoRoot}\packaging\windows\config.example.toml"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\Lumen"; Filename: "{app}\lumen.exe"
Name: "{autodesktop}\Lumen"; Filename: "{app}\lumen.exe"; Tasks: desktopicon

[Tasks]
Name: "desktopicon"; Description: "Create a &desktop shortcut"; Flags: unchecked

[Run]
Filename: "{app}\lumen.exe"; Description: "Launch Lumen"; Flags: nowait postinstall skipifsilent

[Code]
// Lumen's windows are WebView2. It ships with Windows 11 and current Windows 10; warn (do
// not fail) when the runtime is missing so the operator knows why the windows would be blank.
function WebView2Installed(): Boolean;
var
  v: String;
begin
  Result := RegQueryStringValue(HKLM, 'SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}', 'pv', v)
         or RegQueryStringValue(HKCU, 'SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}', 'pv', v);
  Result := Result and (v <> '') and (v <> '0.0.0.0');
end;

function InitializeSetup(): Boolean;
begin
  Result := True;
  if not WebView2Installed() then
    if MsgBox('The Microsoft WebView2 runtime was not found. Lumen needs it to show its windows.' + #13#10 +
              'Install it from https://developer.microsoft.com/microsoft-edge/webview2/ and then continue.' + #13#10#13#10 +
              'Continue installing Lumen anyway?', mbConfirmation, MB_YESNO) = IDNO then
      Result := False;
end;
