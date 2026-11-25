# DEBUG (win-copy-flake): TestContentWindowWebViewEditMenu alone, looped in
# one process per arm, with the clipboard sampler of
# internal/e2e/copyprobe_windows_test.go. Arms: baseline, then with Windows'
# text input service and ctfmon stopped. Fails the step at the end, so the
# job's other steps are skipped and the callbacks are printed.
$ErrorActionPreference = 'Continue'
$env:MYGO_E2E = '1'
$n = 150
$run = '^TestContentWindowWebViewEditMenu$'

function Arm([string]$name) {
  "===== DEBUG arm: $name ($n)"
  "===== DEBUG arm start: $name" | Out-File -Append -Encoding utf8 $env:MYGO_E2E_CALLBACKS_FILE
  $f = Join-Path $env:RUNNER_TEMP "arm-$name.txt"
  go test -count=$n -run $run -timeout 40m ./internal/e2e 2>&1 | Tee-Object -FilePath $f
  $fails = @(Select-String -Path $f -Pattern '--- FAIL').Count
  "===== DEBUG arm: $name | FAIL $fails of $n"
  "===== DEBUG arm end: $name | FAIL $fails of $n" | Out-File -Append -Encoding utf8 $env:MYGO_E2E_CALLBACKS_FILE
}

function Suspects {
  Get-Service TextInputManagementService -ErrorAction SilentlyContinue | Format-Table -AutoSize Name, Status, StartType | Out-String
  Get-Process ctfmon, TextInputHost, rdpclip -ErrorAction SilentlyContinue | Format-Table -AutoSize Name, Id, SessionId, StartTime | Out-String
  "session: $env:SESSIONNAME"
}

Suspects
Arm baseline
"-- DEBUG: stop TextInputManagementService, ctfmon and TextInputHost"
Stop-Service TextInputManagementService -Force -ErrorAction SilentlyContinue
Set-Service TextInputManagementService -StartupType Disabled -ErrorAction SilentlyContinue
Get-Process ctfmon, TextInputHost -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Suspects
Arm no-text-input
"-- DEBUG: after the arm (did they come back?)"
Suspects
exit 1
