# Releasing ZVM

Releasing ZVM is meant to be simple. There are only 5 steps.

1. Update the `<version>` in `cli/meta/version.go`
2. Create a new `git tag <version>`
3. Push the code to said version `git push origin <version>`
4. Build the releases with `deno task build`
5. Draft a new release on GitHub and upload the archives

The Claude Code plugin under `.claude-plugin/` and `skills/` syncs itself. The `Plugin Sync` workflow runs on release, regenerates the skill's command reference from the CLI, and matches the plugin version to `cli/meta/version.go`. Drift is normally caught earlier: `go test ./...` fails on a stale reference, so CI flags it at pull-request time. Regenerate locally with `go test -run TestSkillReference -update .`.
