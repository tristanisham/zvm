# Releasing ZVM

Releasing ZVM is meant to be simple. There are only 5 steps.

1. Update the `<version>` in `cli/meta/version.go`
2. Create a new `git tag <version>`
3. Push the code to said version `git push origin <version>`
4. Build the releases with `deno task build`
5. Draft a new release on GitHub and upload the archives