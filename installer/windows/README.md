# Windows MSI and Winget

The `winget.yml` workflow builds per-user x64 and ARM64 MSI packages on
`windows-latest`. Each package installs `zvm.exe` to
`%USERPROFILE%\.zvm\self`, sets `ZVM_INSTALL`, and adds both
`%USERPROFILE%\.zvm\self` and `%USERPROFILE%\.zvm\bin` to the user's `PATH`.
The packaged binary is built with the `noAutoUpgrades` tag so updates remain
managed by Winget.

The installer project intentionally pins WiX 5.0.2. WiX 6 and later binary
releases are distributed under Open Source Maintenance Fee terms.

Publishing a non-prerelease GitHub release attaches both MSI files to the
release. Once `TristanIsham.ZVM` exists in `microsoft/winget-pkgs`, the workflow
also uses WingetCreate to open the version-update pull request.

## Repository setup

Add a repository secret named `WINGET_CREATE_GITHUB_TOKEN`. It must contain a
GitHub personal access token (classic). WingetCreate's CI documentation asks
for the `repo` scope; this repository's secret is scoped to `public_repo`
instead, on the reasoning that the token only forks `microsoft/winget-pkgs` and
pushes a branch, and both repositories are public. If an automated submission
ever fails on permissions, widen the scope to `repo`. If the secret is absent,
MSI creation and release uploads still run, but Winget submission is skipped.

To rotate the token, create a replacement and overwrite the secret in place:

```bash
gh secret set WINGET_CREATE_GITHUB_TOKEN --repo tristanisham/zvm
```

## First Winget submission

WingetCreate's `new` command is interactive, so the first package submission is
a one-time maintainer step after publishing a release:

```powershell
Invoke-WebRequest https://aka.ms/wingetcreate/latest -OutFile wingetcreate.exe
.\wingetcreate.exe new `
  https://github.com/tristanisham/zvm/releases/download/v0.8.29/zvm-windows-amd64.msi `
  https://github.com/tristanisham/zvm/releases/download/v0.8.29/zvm-windows-arm64.msi
```

Set the package identifier to `TristanIsham.ZVM`, review the generated
manifests, and submit the pull request. After Microsoft accepts it, future
non-prerelease releases are submitted automatically by the workflow.
