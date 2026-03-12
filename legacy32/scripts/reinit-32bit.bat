@echo off
if "%PROCESSOR_ARCHITECTURE%"=="AMD64" goto relaunch32

echo Already in 32-bit context (or 32-bit OS).
echo Running legacy init...
:: your commands here
"C:\Path\To\32bitTool.exe" /reinitialize
goto end

:relaunch32
echo Relaunching in 32-bit mode...
%windir%\SysWOW64\cmd.exe /c "%~f0" %*
exit

:end