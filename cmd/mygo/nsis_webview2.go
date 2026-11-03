package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	fetch "github.com/egoist/mygo/internal/download"
	"github.com/egoist/mygo/internal/webview2"
)

// windows.webview2 says what the installer does about the WebView2
// Runtime, which an app of web views needs: "download" (the default)
// fetches Microsoft's bootstrapper when the machine has no runtime, "embed"
// carries it in the installer (some 1.8 MB), and "none" leaves the runtime
// alone, as an app of native UI wants. Any failure, in the network, the
// signature or the bootstrapper, shows a message with the download link and
// the app is installed all the same: the app's own message tells the user
// at run time (internal/windows).
const (
	webView2Download = "download"
	webView2Embed    = "embed"
	webView2None     = "none"
)

// webView2Client is the key under EdgeUpdate\Clients where the runtime
// records its version, "pv"; tests name another, so that a machine's
// runtime does not hide the installer's paths.
var webView2Client = "{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}"

// webView2URL is where the bootstrapper comes from, in the installer
// (download) and in the build (embed); tests name a server of their own.
var webView2URL = webview2.Download

// webView2PS1 downloads the bootstrapper to $env:MYGO_WV2_FILE unless
// $env:MYGO_WV2_URL is empty (embed), checks its Authenticode signature,
// which must be valid and Microsoft Corporation's, since the evergreen
// file has no fixed hash, and runs it. It prints why it failed and exits
// with 1 (download), 2 (signature), 3 (signer) or 4 (the bootstrapper).
const webView2PS1 = `$ErrorActionPreference = 'Stop'
$file = $env:MYGO_WV2_FILE
if ($env:MYGO_WV2_URL) {
	try {
		[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
		$ProgressPreference = 'SilentlyContinue'
		Invoke-WebRequest -UseBasicParsing -Uri $env:MYGO_WV2_URL -OutFile $file -TimeoutSec 120
	} catch { Write-Output ('download failed: ' + $_.Exception.Message); exit 1 }
}
$sig = Get-AuthenticodeSignature -LiteralPath $file
if ($sig.Status -ne 'Valid') { Write-Output ('signature ' + $sig.Status); exit 2 }
$signer = $sig.SignerCertificate.GetNameInfo('SimpleName', $false)
if ($signer -ne 'Microsoft Corporation') { Write-Output ('signed by ' + $signer); exit 3 }
$p = Start-Process -FilePath $file -ArgumentList '/silent', '/install' -Wait -PassThru
if ($p.ExitCode -ne 0) { Write-Output ('bootstrapper exit code ' + $p.ExitCode); exit 4 }
exit 0
`

// nsisWriteLines returns the NSIS commands that write text to the file
// that the file handle $2 opened, line by line: NSIS strings hold 1024
// characters.
func nsisWriteLines(text string) string {
	var b strings.Builder
	for _, line := range strings.SplitAfter(text, "\n") {
		if line != "" {
			b.WriteString("  FileWrite $2 " + nsisString(strings.TrimSuffix(line, "\n")+"\r\n") + "\n")
		}
	}
	return b.String()
}

// nsisWebView2 returns the installer's function EnsureWebView2 and the
// command that calls it at the end of the install, or "" for both with
// windows.webview2 "none". It does nothing when the registry names a
// runtime, in HKLM (both views) or HKCU. Otherwise it gets the
// bootstrapper (webView2PS1) from the file c.webView2Setup, which the
// installer carries, or from webView2URL, and writes its outcome to
// $TEMP\<identifier>-webview2.log. A failure shows a message with the
// download link, which a silent install (/S) leaves out.
func nsisWebView2(c *Config) (function, call string) {
	if c.Windows.WebView2 == webView2None {
		return "", ""
	}
	key := nsisString(`Software\Microsoft\EdgeUpdate\Clients\` + webView2Client)
	log := `"$TEMP\` + nsisEscape(fsName(c.Identifier)) + `-webview2.log"`
	// Reads pv from root, and goes to found with the version unless it is empty or zero.
	read := func(view, root string) string {
		return `  SetRegView ` + view + `
  ReadRegStr $0 ` + root + ` ` + key + ` "pv"
  StrCmp $0 "" +3
  StrCmp $0 "0.0.0.0" +2
  Goto found
`
	}
	get := `  System::Call 'Kernel32::SetEnvironmentVariable(t "MYGO_WV2_URL", t ` + nsisString(webView2URL) + `)'
`
	if c.Windows.WebView2 == webView2Embed {
		get = `  SetOutPath "$PLUGINSDIR"
  File "/oname=wv2setup.exe" ` + nsisString(c.webView2Setup) + `
  SetOutPath "$INSTDIR"
  System::Call 'Kernel32::SetEnvironmentVariable(t "MYGO_WV2_URL", t "")'
`
	}
	command := `'"$1" -NoProfile -NonInteractive -ExecutionPolicy Bypass -Command "& ([scriptblock]::Create([IO.File]::ReadAllText($$env:MYGO_WV2_SCRIPT)))"'`
	// $0 holds the reason, which NSIS reads inside the string.
	message := `"` + nsisEscape(c.windowsName()+" needs the Microsoft Edge WebView2 Runtime, and the installer could not install it (") + "$0" +
		nsisEscape("). The app is installed, but it shows no page until the runtime is.\n\nOpen the download page?") + `"`
	function = `; Installs the WebView2 Runtime when the machine has none (windows.webview2).
Function EnsureWebView2
  Push $0
  Push $1
  Push $2
` + read("32", "HKLM") + read("64", "HKLM") + read("64", "HKCU") + `  SetRegView 32
  InitPluginsDir
  StrCpy $1 "$WINDIR\Sysnative\WindowsPowerShell\v1.0\powershell.exe"
  IfFileExists $1 +2
  StrCpy $1 "$SYSDIR\WindowsPowerShell\v1.0\powershell.exe"
  IfFileExists $1 +3
  StrCpy $0 "PowerShell is missing"
  Goto failed
  DetailPrint "Installing the Microsoft Edge WebView2 Runtime"
  FileOpen $2 "$PLUGINSDIR\webview2.ps1" w
` + nsisWriteLines(webView2PS1) + `  FileClose $2
  System::Call 'Kernel32::SetEnvironmentVariable(t "MYGO_WV2_SCRIPT", t "$PLUGINSDIR\webview2.ps1")'
  System::Call 'Kernel32::SetEnvironmentVariable(t "MYGO_WV2_FILE", t "$PLUGINSDIR\wv2setup.exe")'
` + get + `  nsExec::ExecToStack ` + command + `
  Pop $2
  Pop $0
  StrCmp $2 "0" installed
  failed:
  FileOpen $2 ` + log + ` w
  FileWrite $2 "WebView2 Runtime not installed: $0$\r$\n"
  FileClose $2
  MessageBox MB_YESNO|MB_ICONEXCLAMATION ` + message + ` /SD IDNO IDNO done
  ExecShell "open" ` + nsisString(webview2.Download) + `
  Goto done
  installed:
  FileOpen $2 ` + log + ` w
  FileWrite $2 "WebView2 Runtime installed$\r$\n"
  FileClose $2
  Goto done
  found:
  FileOpen $2 ` + log + ` w
  FileWrite $2 "WebView2 Runtime found: $0$\r$\n"
  FileClose $2
  SetRegView 32
  done:
  Pop $2
  Pop $1
  Pop $0
FunctionEnd

`
	return function, "  Call EnsureWebView2\n"
}

// fetchWebView2Setup downloads the bootstrapper into dir for the embed
// mode, and returns its path. Tests replace it.
var fetchWebView2Setup = func(dir string) (string, error) {
	t := con.start("Downloading the WebView2 Runtime's bootstrapper for the Windows installer")
	path := filepath.Join(dir, "MicrosoftEdgeWebview2Setup.exe")
	err := fetch.Get(context.Background(), downloadFile(t, webView2URL, ""), path)
	if t.end(err, "Downloaded the WebView2 Runtime's bootstrapper") != nil {
		return "", fmt.Errorf("windows.webview2 is embed, and downloading the bootstrapper failed (use download, or build with -skip-installer):\n%w", err)
	}
	if info, err := os.Stat(path); err != nil || info.Size() == 0 {
		return "", fmt.Errorf("the WebView2 bootstrapper downloaded from %s is empty", webView2URL)
	}
	return path, nil
}
