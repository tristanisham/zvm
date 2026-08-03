# Homebrew

`zvm.rb` is ZVM's Homebrew formula, kept here as the submission source for
`Homebrew/homebrew-core`.

Nothing in this repository builds or consumes this file. It is a reference copy.
**Once the formula is merged into `homebrew-core`, that copy becomes canonical** and
Homebrew maintainers own it. Treat this file as a starting point for the initial
submission, not as a mirror to keep in sync.

The formula builds from source with `-tags noAutoUpgrades`, so `zvm upgrade` is compiled
out and Homebrew owns version management. It does not use the `.tar` release assets that
`build.ts` produces; `url` points at GitHub's automatically generated source archive for
the tag.

## Why `homebrew-core` and not a tap

Since Homebrew 6.0.0, third-party taps require explicit trust, so installing by short name
from a tap costs three commands (`brew tap`, `brew trust --formula`, `brew install`).
Core needs none of that — just `brew install zvm`.

Core formulae are also autobumped by BrewTestBot, so new releases get version-bump pull
requests automatically. That removes the release workflow, access token and second
repository a tap would have required.

## Prerequisites

Homebrew must be installed. Homebrew on Linux is sufficient — `homebrew-core` CI covers
macOS and Linux itself.

## Updating the version and checksum

`url` and `sha256` must point at the release being submitted:

```sh
VERSION=0.8.29
curl -sL -o zvm-src.tar.gz \
  "https://github.com/tristanisham/zvm/archive/refs/tags/v${VERSION}.tar.gz"
sha256sum zvm-src.tar.gz
```

## Submitting to homebrew-core

This is a one-time step. After it merges, autobump handles subsequent releases.

1. Fork [Homebrew/homebrew-core](https://github.com/Homebrew/homebrew-core) on GitHub.

2. Set up a contribution checkout:

   ```sh
   brew tap --force homebrew/core
   cd "$(brew --repository homebrew/core)"
   git remote add tristanisham https://github.com/tristanisham/homebrew-core.git
   git checkout -b zvm origin/HEAD
   ```

3. Copy the formula into place:

   ```sh
   cp /path/to/zvm/installer/brew/zvm.rb Formula/z/zvm.rb
   ```

4. Validate. All of these must pass before submitting:

   ```sh
   HOMEBREW_NO_INSTALL_FROM_API=1 brew install --build-from-source zvm
   brew test zvm
   brew audit --strict --new --online zvm
   brew style --fix --formula zvm
   brew lgtm --online
   ```

5. Commit and open the pull request. New formulae use the
   `<name> <version> (new formula)` title convention:

   ```sh
   git add Formula/z/zvm.rb
   git commit -m "zvm 0.8.29 (new formula)"
   git push tristanisham zvm
   ```

## What to expect

Review is volunteer-driven, so days to weeks is normal. Likely review notes are the
caveats wording and the test block.

After merge, BrewTestBot builds bottles, `brew install zvm` works with no tap and no trust
step, and future releases get automatic version-bump pull requests.

If the submission is rejected, the fallback is a third-party tap at
`tristanisham/homebrew-zvm`, using a near-identical formula. That costs users the trust
step described above, and costs us a release workflow to bump the formula, since autobump
only applies to official repositories.
