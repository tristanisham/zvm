# Contributing

Thank you for contributing to ZVM. I'm just one guy, so all contributions are
welcome and appreciated. However, in order to save us all some time, I've
decided to write up some basic rules for contributing that I'd like all future
P.R.'s to follow.

1. Please run `go fmt .` before submitting your P.R. Go is a very opinionated
   language, and how you define function can affect visibilty across the whole
   program. Unless your function needs to be used by a different package,
   default to private (lowercase names). You should also use camelCase for all
   your variables, functions, and fields. Submissions using snake case, kebab
   case, or any other case will be refused until adjusted to camel case.

2. I have a lot of obligations besides this repository so P.R.s may languish for
   a few days or weeks while I deal with my external obligations. It is
   important to me that you know that this is a very actively developed
   repository and all P.R.s will be reviewed within a month of submission.
   Probably sooner. If you would like to talk to me about your P.R. feel free to
   [email](mailto:tristan.isham@hey.com) me or reach out on Discord or Twitter.
   I check all regularly.

I hope you feel comfortable contributing to ZVM and know that I greatly
appreaciate every contribution. I'm just one person so seeing community members
care enough to write code for this project warms my heart. Thank you in advance.

## Validation

When reviewing or changing Go code, run `go test ./...`, `go vet ./...`, and
`govulncheck ./...`. Include the results in your P.R. If you don't have
`govulncheck`, install it with
`go install golang.org/x/vuln/cmd/govulncheck@latest`.

## Signing a release

CI builds the archives, checksums, and GitHub provenance attestations. You can
also sign `SHA256SUMS` locally with [Minisign](https://jedisct1.github.io/minisign/).
The signed checksums cover every release archive.

Install Minisign and generate a key pair once:

```sh
minisign -G
```

This writes `minisign.pub` in the current directory and an encrypted private
key at `~/.minisign/minisign.key`. Back up the private key and password. Publish
the public key on your website or in the repository so users can verify your
signatures. Keep the private key out of the repository and CI.

After CI creates the draft release, sign it before publishing:

```sh
deno task release:sign          # tag from cli/meta/version.go
deno task release:sign v0.9.5   # or an explicit tag
```

The task downloads the draft's `SHA256SUMS`, prompts for your key password,
signs it, checks the signature against `minisign.pub`, and uploads
`SHA256SUMS.minisig`. It refuses releases that are already published. For a
key stored elsewhere, set `MINISIGN_CONFIG_DIR` to the directory holding
`minisign.key`. If you rebuild or replace the draft's archives, run it again.
See `RELEASING.md` for details.

To verify a download, put the archives, `SHA256SUMS`, and `SHA256SUMS.minisig`
in the same directory. Use the public key you published:

```sh
minisign -V -m SHA256SUMS -p /path/to/minisign.pub
sha256sum --check --ignore-missing SHA256SUMS
```

Run the checksum check only after the signature verifies. The install scripts
and `zvm upgrade` don't verify these Minisign signatures yet.

## AI

Regardless of your thoughts on generative AI, as
[Linus Torvalds said](https://www.youtube.com/watch?v=mfv0V1SxbNA), "The genie
is out of the bottle". ZVM itself employs Gemini Code Assist in this repo to
assist with evaluating new PRs for vulnerabilies, styling defects, and other
issues that can be address by the pull request's author. It does not make any
decisions or write any code without manual review and approval.

If you are submitting a pull request to ZVM that was written with generative AI,
I ask you to follow these rules, or have your P.R. rejected and future
contributions sidelined.

### 1. Know what the AI wrote.

If you cannot explain, identify, or defend the choices and code written by your
AI agent, you should not be making pull request. ZVM is a tool installed on
thousands of computers. That is, in part, due to the trust our users have in the
quality and security of the application. **I do not take that lightly**. AI
cannot be held responsible for decisions it makes, but you will be. This is not
a vibe-coded app, and contributors are expected to have the requisite skills
that, if AI were not to exist, write the code themselves, or at least write it
in a language they are already familiar with.

### 2. Clearly Disclose AI usage

P.R.s that are written with Generative AI and that do not disclose it will not
be merged. If it is found out that generative AI was used and not disclosed in a
pull request, your future contributions will be sidelined.

### 3. Do not add AI metadata in your P.R.

P.R.s that add AI metadata such as `AGENT.md` or `Claude.md` will not be merged.
