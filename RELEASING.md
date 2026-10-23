# Releasing ZVM

Releasing ZVM is meant to be simple. CI builds the archives; you tag, sign,
and publish.

1. Update the `<version>` in `cli/meta/version.go`, run `deno task pkg`, and
   commit the synchronized package metadata
2. Create a new `git tag <version>`
3. Push the code to said version `git push origin <version>`. The `Release`
   workflow builds every archive, writes `SHA256SUMS`, attests build
   provenance, and uploads them to a **draft** release. It fails if the tag
   doesn't match `VERSION`.
4. Sign the draft's checksums with Minisign (see below)
5. Review the draft's notes on GitHub and publish it. Publishing stays manual
   because a release published by the workflow's token wouldn't trigger
   `winget.yml`.

## Signing

`SHA256SUMS` covers every archive, so signing it signs the whole release. The
private key lives only on the maintainer's machine, never in CI; the public key
is `minisign.pub` and is also printed in the README.

After the workflow finishes, sign the checksum file it uploaded, not one from a
local build, which can differ:

```sh
tag=<version>
gh release download "$tag" --repo tristanisham/zvm --pattern SHA256SUMS --dir build --clobber
deno task release:sign -t "ZVM $tag release checksums"
minisign -Vm build/SHA256SUMS -p minisign.pub
gh release upload "$tag" build/SHA256SUMS.minisig --repo tristanisham/zvm --clobber
```

Sign last. Re-running the workflow on a draft replaces `SHA256SUMS`, which
invalidates an uploaded signature; download and sign again before publishing.

First-time key setup and backup are covered in `CONTRIBUTING.MD`. If the key is
ever lost or rotated, update both `minisign.pub` and the key in the README.

`pkg.ts` keeps `flake.nix` and the Claude/Codex plugin manifests aligned with
`cli/meta/version.go`. `deno task build` runs it automatically, but run
`deno task pkg` before tagging so those changes are included in the release
commit. Use `deno task pkg:check` to verify that the versions already match.

The `Plugin Sync` workflow is a release-time backstop and also regenerates the
skill's command reference from the CLI. Drift is normally caught earlier by
`go test ./...`. Regenerate the reference locally with
`go test -run TestSkillReference -update .`.
