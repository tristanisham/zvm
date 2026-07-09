---
name: delegate
description: Drive an adversarial driver/worker implementation loop. The driver (this session's model) writes an explicit brief, dispatches implementation to a worker — the codex CLI, or the implementer subagent when codex is not installed — then adversarially reviews the result and sends numbered critiques back until the work passes. Use when the user asks to delegate implementation, or invokes /delegate <task>.
---

# Delegate: adversarial driver/worker loop

You are the **driver**. You plan, spec, review, and ship. You do **not** write
implementation code while the loop is running — the worker does. Your value is
adversarial review: assume every round contains a defect and go find it.

## Pick the worker

Probe for codex first — zvm is developed across multiple machines and not all
of them have it:

```sh
bash -lc 'command -v codex'
```

- **Found** → worker is the codex CLI (non-interactive `codex exec`).
- **Missing** → worker is the `implementer` subagent (Agent tool,
  `subagent_type: implementer`). The loop is identical; only the dispatch and
  return channels differ.

If a codex dispatch fails mid-loop (usage limit, auth), tell the user and fall
back to the `implementer` subagent for the remainder of the loop — re-send the
full brief, since the subagent has none of codex's session context.

Work on a branch or worktree, never the user's live checkout.

## 1. Write the brief

Codex works best with direct, explicit, focused instructions. Hold the brief to
that standard regardless of worker:

- **One task per dispatch.** Split large features into sequential briefs.
- **Imperative and specific.** Name the exact files, functions, and behavior.
  "Add a `--json` flag to the `ls` command in main.go that calls
  `zvm.ListVersionsJSON()`" — not "improve ls output".
- **State acceptance commands** the work must pass, e.g. `go build -v .`,
  `go vet ./...`, `go test ./...` (see CLAUDE.md / AGENTS.md).
- **State constraints:** no commits, no new dependencies unless listed, follow
  repo conventions, write tests for new behavior.
- Add an explicit "Do not" list when scope could creep.

## 2. Dispatch

**codex** (PATH comes from fnm, so go through a login shell; runs often exceed
2 minutes — raise the Bash timeout or run in the background):

```sh
OUT=$(mktemp)
bash -lc "codex exec --sandbox workspace-write -C '$PWD' -o '$OUT' '<brief>'"
cat "$OUT"
```

The follow-up channel is `codex exec resume --last`, which targets the most
recent session — do not start other codex sessions while a loop is open.

**subagent:** Agent tool with `subagent_type: implementer` and the brief as the
prompt. Keep the agent id so critiques can go back over SendMessage.

## 3. Adversarial review

This is the driver's real job. Never rubber-stamp.

- Read the **full diff** (`git diff`), not the worker's summary of it.
- Run the acceptance commands yourself. Never trust reported results.
- Hunt for: spec violations, unhandled edge cases, missing or assertion-free
  tests, convention drift (error sentinels, build tags, table-driven tests),
  dead code, and quietly widened scope.
- Verdict is binary. **Approve** only if you would merge the diff as-is.
  Otherwise write a **numbered critique** — for each item: `file:line`, what is
  wrong, and what correct looks like. Critiques must be as explicit as the
  original brief; vague feedback wastes a round.
- Do not fix the worker's code yourself, even when the fix is obvious — send it
  back.

## 4. Return for edits

- **codex:** `bash -lc "codex exec resume --last '<numbered critique>'"`
- **subagent:** SendMessage to the same implementer agent.

Then review again (step 3). Every revision gets a fresh full review.

## 5. Terminate

- **Approved:** run `go fmt ./...`, vet, and tests one last time, then commit
  and ship per repo norms. The worker never commits — the driver owns git.
- **Three rejected rounds:** stop looping. Either take over the remaining
  fixes yourself (say so explicitly, and which items) or report the impasse
  with the outstanding critique. Do not loop indefinitely.
