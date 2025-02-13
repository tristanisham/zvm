
function Install-ZVM {
    param(
        [string]$urlSuffix
    );

    $TempDir = Join-Path $env:TEMP "zvm-install"
    New-Item -ItemType Directory -Force -Path $TempDir | Out-Null
    Remove-Item -Path $TempDir -Force -Recurse -ErrorAction SilentlyContinue
    $Target = $urlSuffix
    $URL = "https://github.com/tristanisham/zvm/releases/latest/download/$urlSuffix"
    $ZipPath = Join-Path $TempDir $Target

    New-Item -ItemType Directory -Force -Path $TempDir | Out-Null
    # Check if ZVM_BINARY_ON_DISK_LOCATION is set
    if ($env:ZVM_BINARY_ON_DISK_LOCATION) {
        # Create a zip file from the existing binary
        $sourceExe = Join-Path $env:ZVM_BINARY_ON_DISK_LOCATION "zvm.exe"
        if (Test-Path $sourceExe) {
            $tempZipDir = Join-Path $TempDir "zvm-windows-$PROCESSOR_ARCH"
            New-Item -ItemType Directory -Force -Path $tempZipDir | Out-Null
            Copy-Item $sourceExe -Destination $tempZipDir
            Compress-Archive -Path $tempZipDir -DestinationPath $ZipPath -Force
        }
        else {
            Write-Output "ZVM_BINARY_ON_DISK_LOCATION is set but zvm.exe not found at $sourceExe"
            exit 1
        }
    }
    else {
        # Original download logic
        $URL = "https://github.com/tristanisham/zvm/releases/latest/download/$urlSuffix"
        Remove-Item -Force $ZipPath -ErrorAction SilentlyContinue
        curl.exe "-#SfLo" "$ZipPath" "$URL"
        if ($LASTEXITCODE -ne 0) {
            Write-Output "Install Failed - could not download $URL"
            Write-Output "The command 'curl.exe $URL -o $ZipPath' exited with code ${LASTEXITCODE}`n"
            exit 1
        }
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
        Expand-Archive "$ZipPath" "$TempDir" -Force
        $global:ProgressPreference = $lastProgressPreference
        if (!(Test-Path "${TempDir}\$UnzippedPath\zvm.exe")) {
            throw "The file '${TempDir}\$UnzippedPath\zvm.exe' does not exist. Download is corrupt / Antivirus intercepted?`n"
        }
    }
    catch {
        Write-Output "Install Failed - could not unzip $ZipPath"
        Write-Error $_
        exit 1
    }
    Remove-Item "${ZVMSelf}\zvm.exe" -ErrorAction SilentlyContinue
    # Run the temporary binary to get installation paths
    $EnvJson = & "${TempDir}\$UnzippedPath\zvm.exe" env | ConvertFrom-Json
    $ZVMSelf = New-Item -ItemType Directory -Force -Path $EnvJson.self
    $ZVMBin = New-Item -ItemType Directory -Force -Path $EnvJson.bin
    Move-Item "${TempDir}\$UnzippedPath\zvm.exe" "$($EnvJson.self)\zvm.exe" -Force

    Remove-Item "${ZipPath}" -Recurse -Force
    Remove-Item ${TempDir}\$UnzippedPath -Force

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
    Write-Output "The binary is located at $($EnvJson.self)\zvm.exe`n"

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

    Write-Output "To get started, restart your terminal/editor, then type `"zvm`"`n"
}


$PROCESSOR_ARCH = $env:PROCESSOR_ARCHITECTURE.ToLower()
Install-ZVM "zvm-windows-$PROCESSOR_ARCH.zip"
