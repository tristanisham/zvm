# Releasing ZVM

Releasing ZVM is meant to be simple. CI builds the archives; you tag
and publish.

1. Update the `<version>` in `cli/meta/version.go`, run `deno task pkg`, and
   commit the synchronized package metadata
2. Create a new `git tag <version>`
3. Push the code to said version `git push origin <version>`. The `Release`
   workflow builds every archive, writes `SHA256SUMS`, attests build
   provenance, and uploads them to a **draft** release. It fails if the tag
   doesn't match `VERSION`.
4. Review the draft's notes on GitHub and publish it. Publishing stays manual
   because a release published by the workflow's token wouldn't trigger
   `winget.yml`.

`.github/scripts/pkg.ts` keeps `flake.nix` and the Claude/Codex plugin
manifests aligned with `cli/meta/version.go`. `deno task build` runs it
automatically, but run `deno task pkg` before tagging so those changes are
included in the release commit. Use `deno task pkg:check` to verify that the
versions already match.

`deno task release` runs the package tests, then the build tests, checks package
versions, and builds the release archives. It stops if any step fails. The
`Release` workflow uses this task; you can also run it locally before tagging.

The `Plugin Sync` workflow is a release-time backstop and also regenerates the
skill's command reference from the CLI. Drift is normally caught earlier by
`go test ./...`. Regenerate the reference locally with
`go test -run TestSkillReference -update .`.
