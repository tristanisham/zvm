// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

import { join } from "node:path";
import { REPO, signRelease } from "./release_sign.ts";

// Fake gh and minisign log their arguments; RELEASE_STATE is what
// `gh release view` reports, or "missing" for no release at all.
async function fixture(): Promise<{ root: string; log: string }> {
  const root = await Deno.makeTempDir();
  const bin = join(root, "bin");
  const log = join(root, "calls");
  await Deno.mkdir(join(root, "cli", "meta"), { recursive: true });
  await Deno.writeTextFile(
    join(root, "cli", "meta", "version.go"),
    'package meta\n\nconst (\n\tVERSION = "v1.2.3"\n)\n',
  );
  await Deno.mkdir(bin);
  for (const name of ["gh", "minisign"]) {
    await Deno.writeTextFile(
      join(bin, name),
      `#!/bin/sh\necho "${name} $*" >> "${log}"\n` +
        (name === "gh"
          ? '[ "$2" != view ] && exit 0\n[ "$RELEASE_STATE" = missing ] && exit 1\necho "$RELEASE_STATE"\n'
          : ""),
    );
    await Deno.chmod(join(bin, name), 0o755);
  }
  return { root, log };
}

async function withState(
  state: string,
  root: string,
  run: () => Promise<void>,
): Promise<void> {
  const path = Deno.env.get("PATH") ?? "";
  Deno.env.set("PATH", `${join(root, "bin")}:${path}`);
  Deno.env.set("RELEASE_STATE", state);
  try {
    await run();
  } finally {
    Deno.env.set("PATH", path);
    Deno.env.delete("RELEASE_STATE");
  }
}

async function calls(log: string): Promise<string[]> {
  try {
    return (await Deno.readTextFile(log)).trim().split("\n");
  } catch (err) {
    if (err instanceof Deno.errors.NotFound) return [];
    throw err;
  }
}

Deno.test({
  name: "release:sign downloads, signs, verifies and uploads a draft",
  ignore: Deno.build.os === "windows",
  async fn() {
    const { root, log } = await fixture();
    try {
      await withState("true", root, async () => {
        if (await signRelease({ root }) !== "v1.2.3") {
          throw new Error("tag was not read from version.go");
        }
      });
      const repo = `--repo ${REPO}`;
      const want = [
        `gh release view v1.2.3 ${repo} --json isDraft --jq .isDraft`,
        `gh release download v1.2.3 ${repo} --pattern SHA256SUMS --dir build --clobber`,
        "minisign -S -m build/SHA256SUMS -t ZVM v1.2.3 release checksums",
        "minisign -V -m build/SHA256SUMS -p minisign.pub",
        `gh release upload v1.2.3 ${repo} build/SHA256SUMS.minisig --clobber`,
      ];
      const got = await calls(log);
      if (JSON.stringify(got) !== JSON.stringify(want)) {
        throw new Error(`calls:\n${got.join("\n")}\nwant:\n${want.join("\n")}`);
      }
    } finally {
      await Deno.remove(root, { recursive: true });
    }
  },
});

Deno.test({
  name: "release:sign refuses published and missing releases",
  ignore: Deno.build.os === "windows",
  async fn() {
    for (const state of ["false", "missing"]) {
      const { root, log } = await fixture();
      try {
        let failed = false;
        await withState(state, root, async () => {
          try {
            await signRelease({ root, tag: "v9.9.9" });
          } catch {
            failed = true;
          }
        });
        const got = await calls(log);
        if (!failed || got.length !== 1) {
          throw new Error(`${state}: release was touched: ${got.join("; ")}`);
        }
      } finally {
        await Deno.remove(root, { recursive: true });
      }
    }
  },
});
