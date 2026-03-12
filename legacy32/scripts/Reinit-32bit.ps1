# Reinit-32bit.ps1
# Purpose: Relaunch self/script in true 32-bit context if running under 64-bit PowerShell

if ([IntPtr]::Size -eq 8) {
    Write-Host "Currently running as 64-bit PowerShell. Relaunching in 32-bit..."
    $env:PROCESSOR_ARCHITECTURE = "x86"   # hint (not always enough)
    $env:PROCESSOR_ARCHITEW6432   = "AMD64"  # WOW64 marker

    # Use SysWOW64 version of PowerShell
    $ps32 = "$env:windir\SysWOW64\WindowsPowerShell\v1.0\powershell.exe"
    if (Test-Path $ps32) {
        $argList = $MyInvocation.Line -replace [regex]::Escape($PSCommandPath), ''
        Start-Process -FilePath $ps32 -ArgumentList "-NoProfile -ExecutionPolicy Bypass -File `"$PSCommandPath`" $argList" -Wait
        exit
    } else {
        Write-Error "32-bit PowerShell not found. Cannot reinitialize."
        exit 1
    }
}

Write-Host "Now running in 32-bit context (or native 32-bit OS)."
Write-Host "Put your 32-bit legacy commands / init logic here..."

# Example: call your legacy 32-bit tool
# & \"C:\Path\To\Your\Legacy32bitApp.exe\" /init /reinit

# Or build / run hybrid → pure 32-bit transition steps here
