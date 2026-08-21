# Releasing ZVM

Releasing ZVM is meant to be simple. There are only 5 steps.

1. Update the `<version>` in `cli/meta/version.go`, run `deno task pkg`, and
   commit the synchronized package metadata
2. Create a new `git tag <version>`
3. Push the code to said version `git push origin <version>`
4. Build the releases with `deno task build`
5. Draft a new release on GitHub and upload the archives

`pkg.ts` keeps `flake.nix` and the Claude/Codex plugin manifests aligned with
`cli/meta/version.go`. `deno task build` runs it automatically, but run
`deno task pkg` before tagging so those changes are included in the release
commit. Use `deno task pkg:check` to verify that the versions already match.

The `Plugin Sync` workflow is a release-time backstop and also regenerates the
skill's command reference from the CLI. Drift is normally caught earlier by
`go test ./...`. Regenerate the reference locally with
`go test -run TestSkillReference -update .`.
