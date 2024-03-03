@setlocal
@echo off

%*
if %ERRORLEVEL% LSS 1 goto :EOF

:: The command failed without elevation, try with elevation
set CMD=%*
set APP=%1
start wscript //nologo "%~dpn0.vbs" %*
