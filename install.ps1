
function Install-ZVM {
    param(
        [string]$urlSuffix,
        [switch]$NoEnv
    );

    $ZVMRoot = "${Home}\.zvm"
    $ZVMSelf = mkdir -Force "${ZVMRoot}\self"
    $ZVMBin = mkdir -Force "${ZVMRoot}\bin"
    $Target = $urlSuffix
    $URL = "https://github.com/tristanisham/zvm/releases/latest/download/$urlSuffix"
    $ZipPath = "${ZVMSelf}\$Target"

    $null = mkdir -Force $ZVMSelf
    # curl.exe "-#SfLo" "$ZVMSelf/elevate.cmd" "https://raw.githubusercontent.com/tristanisham/zvm/master/bin/elevate.cmd" -s
    #curl.exe "-#SfLo" "$ZVMSelf/elevate.vbs" "https://raw.githubusercontent.com/tristanisham/zvm/master/bin/elevate.vbs" -s
    Remove-Item -Force $ZipPath -ErrorAction SilentlyContinue
    curl.exe "-#SfLo" "$ZipPath" "$URL" 
    if ($LASTEXITCODE -ne 0) {
        Write-Output "Install Failed - could not download $URL"
        Write-Output "The command 'curl.exe $URL -o $ZipPath' exited with code ${LASTEXITCODE}`n"
        exit 1
    }
    if (!(Test-Path $ZipPath)) {
        Write-Output "Install Failed - could not download $URL"
        Write-Output "The file '$ZipPath' does not exist. Did an antivirus delete it?`n"
        exit 1
    }
    $UnzippedPath = $Target.Substring(0, $Target.Length - 4)
    try {
        $lastProgressPreference = $global:ProgressPreference
        $global:ProgressPreference = 'SilentlyContinue';
        Expand-Archive "$ZipPath" "$ZVMSelf" -Force
        $global:ProgressPreference = $lastProgressPreference
        if (!(Test-Path "${ZVMSelf}\$UnzippedPath\zvm.exe")) {
            throw "The file '${ZVMSelf}\$UnzippedPath\zvm.exe' does not exist. Download is corrupt / Antivirus intercepted?`n"
        }
    }
    catch {
        Write-Output "Install Failed - could not unzip $ZipPath"
        Write-Error $_
        exit 1
    }
    Remove-Item "${ZVMSelf}\zvm.exe" -ErrorAction SilentlyContinue
    Move-Item "${ZVMSelf}\$UnzippedPath\zvm.exe" "${ZVMSelf}\zvm.exe" -Force

    Remove-Item "${ZVMSelf}\$Target" -Recurse -Force
    Remove-Item ${ZVMSelf}\$UnzippedPath -Force

    $null = "$(& "${ZVMSelf}\zvm.exe")"
    if ($LASTEXITCODE -eq 1073741795) {
        # STATUS_ILLEGAL_INSTRUCTION
        Write-Output "Install Failed - zvm.exe is not compatible with your CPU.`n"
        exit 1
    }
    if ($LASTEXITCODE -ne 0) {
        Write-Output "Install Failed - could not verify zvm.exe"
        Write-Output "The command '${ZVMSelf}\zvm.exe' exited with code ${LASTEXITCODE}`n"
        exit 1
    }

    $C_RESET = [char]27 + "[0m"
    $C_GREEN = [char]27 + "[1;32m"

    Write-Output "${C_GREEN}ZVM${DisplayVersion} was installed successfully!${C_RESET}"
    Write-Output "The binary is located at ${ZVMSelf}\zvm.exe`n"

    if (-not $NoEnv) {
        $User = [System.EnvironmentVariableTarget]::User
        $Path = [System.Environment]::GetEnvironmentVariable('Path', $User) -split ';'
        $ZVMInstall = 'ZVM_INSTALL'

        $ZVMInstallValue = [System.Environment]::GetEnvironmentVariable($ZVMInstall, [System.EnvironmentVariableTarget]::User)

        if ($null -eq $ZVMInstallValue) {
            [System.Environment]::SetEnvironmentVariable($ZVMInstall, $ZVMSelf, [System.EnvironmentVariableTarget]::User)
        }

        if ($Path -notcontains $ZVMSelf) {
            $Path += $ZVMSelf
            [System.Environment]::SetEnvironmentVariable('Path', $Path -join ';', $User)
        } 
        if ($env:PATH -notcontains ";${ZVMSelf}") {
            $env:PATH = "${env:Path};${ZVMSelf}"
        }

        if ($Path -notcontains $ZVMBin) {
            $Path += $ZVMBin
            [System.Environment]::SetEnvironmentVariable('Path', $Path -join ';', $User)
        }
        if ($env:PATH -notcontains ";${ZVMBin}") {
            $env:PATH = "${env:Path};${ZVMBin}"
        }
    } else {
        Write-Output "Skipping environment variable setup due to --no-env flag.`n"
    }

    Write-Output "To get started, restart your terminal/editor, then type `"zvm`"`n"
}


$PROCESSOR_ARCH = $env:PROCESSOR_ARCHITECTURE.ToLower()

if ($PROCESSOR_ARCH -eq "x86") {
  Write-Output "Install Failed - ZVM requires a 64-bit environment."
  Write-Output "Please ensure that you are running the 64-bit version of PowerShell or that your system is 64-bit.`n"
  exit 1
}

# Parse --no-env flag if present
$NoEnv = $false
if ($args -contains '--no-env') {
    $NoEnv = $true
}

Install-ZVM "zvm-windows-$PROCESSOR_ARCH.zip" -NoEnv:$NoEnv
