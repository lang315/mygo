# DEBUG (win-copy-flake): TestContentWindowWebViewEditMenu with the clipboard
# sampler of internal/e2e/copyprobe_windows_test.go, in two settings: alone,
# looped in one process, and in its usual place among the package's GUI
# tests, the whole package run as many times as the job allows. Fails the
# step at the end, so the job's other steps are skipped and the callbacks
# are printed.
$ErrorActionPreference = 'Continue'
$env:MYGO_E2E = '1'

function Arm([string]$name, [string[]]$goArgs) {
  "===== DEBUG arm: $name (go test $goArgs)"
  "===== DEBUG arm start: $name" | Out-File -Append -Encoding utf8 $env:MYGO_E2E_CALLBACKS_FILE
  $f = Join-Path $env:RUNNER_TEMP "arm-$name.txt"
  $start = Get-Date
  & go test @goArgs ./internal/e2e 2>&1 | Tee-Object -FilePath $f
  $code = $LASTEXITCODE
  $copy = @(Select-String -Path $f -Pattern '--- FAIL: TestContentWindowWebViewEditMenu').Count
  $all = @(Select-String -Path $f -Pattern '--- FAIL').Count
  $line = "===== DEBUG arm end: $name | exit $code | Copy test FAIL $copy | all FAIL lines $all | took $([int]((Get-Date) - $start).TotalSeconds)s"
  $line
  $line | Out-File -Append -Encoding utf8 $env:MYGO_E2E_CALLBACKS_FILE
}

Arm 'alone' @('-count=100', '-run=^TestContentWindowWebViewEditMenu$', '-timeout=20m')
for ($i = 1; $i -le 3; $i++) {
  Arm "package-$i" @('-count=1', '-timeout=25m')
}
exit 1
