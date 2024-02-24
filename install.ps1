
function Install-ZVM {
    param(
        [string]$urlSuffix
    );

    $ZVMRoot = "${Home}\.zvm"
    $ZVMBin = mkdir -Force "${ZVMRoot}\bin"
    $Target = $urlSuffix
    $URL = "https://github.com/tristanisham/zvm/releases/latest/download/$urlSuffix"
    $ZipPath = "${ZVMBin}\$Target"

    $null = mkdir -Force $ZVMBin
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
        Expand-Archive "$ZipPath" "$ZVMBin" -Force
        $global:ProgressPreference = $lastProgressPreference
        if (!(Test-Path "${ZVMBin}\$UnzippedPath\zvm.exe")) {
            throw "The file '${ZVMBin}\$UnzippedPath\zvm.exe' does not exist. Download is corrupt / Antivirus intercepted?`n"
        }
    }
    catch {
        Write-Output "Install Failed - could not unzip $ZipPath"
        Write-Error $_
        exit 1
    }
    Remove-Item "${ZVMBin}\zvm.exe" -ErrorAction SilentlyContinue
    Move-Item "${ZVMBin}\$UnzippedPath\zvm.exe" "${ZVMBin}\zvm.exe" -Force

    Remove-Item "${ZVMBin}\$Target" -Recurse -Force
    Remove-Item ${ZVMBin}\$UnzippedPath -Force

    $null = "$(& "${ZVMBin}\zvm.exe")"
    if ($LASTEXITCODE -eq 1073741795) {
        # STATUS_ILLEGAL_INSTRUCTION
        Write-Output "Install Failed - zvm.exe is not compatible with your CPU.`n"
        exit 1
    }
    if ($LASTEXITCODE -ne 0) {
        Write-Output "Install Failed - could not verify zvm.exe"
        Write-Output "The command '${ZVMBin}\zvm.exe' exited with code ${LASTEXITCODE}`n"
        exit 1
    }

    $C_RESET = [char]27 + "[0m"
    $C_GREEN = [char]27 + "[1;32m"

    Write-Output "${C_GREEN}ZVM${DisplayVersion} was installed successfully!${C_RESET}"
    Write-Output "The binary is located at ${ZVMBin}\zvm.exe`n"

    $User = [System.EnvironmentVariableTarget]::User
    $Path = [System.Environment]::GetEnvironmentVariable('Path', $User) -split ';'
    if ($Path -notcontains $ZVMBin) {
        $Path += $ZVMBin
        [System.Environment]::SetEnvironmentVariable('Path', $Path -join ';', $User)
    }
    if ($env:PATH -notcontains ";${ZVMBin}") {
        $env:PATH = "${env:Path};${ZVMBin}"
    }
    Write-Output "To get started, restart your terminal/editor, then type `"zvm`"`n"
}


$PROCESSOR_ARCH = $env:PROCESSOR_ARCHITECTURE.ToLower()
Install-ZVM "zvm-windows-$PROCESSOR_ARCH.zip"