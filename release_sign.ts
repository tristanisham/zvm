// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

// Signs a draft release's SHA256SUMS with Minisign and attaches the signature.
//
//   deno task release:sign            # tag from cli/meta/version.go
//   deno task release:sign v0.9.5     # an explicit tag

import { join } from "node:path";
import { packageVersion } from "./pkg.ts";

export const REPO = "tristanisham/zvm";

export interface SignReleaseOptions {
  tag?: string;
  root?: string;
}

async function run(cmd: string, args: string[], cwd: string): Promise<void> {
  const { code } = await new Deno.Command(cmd, {
    args,
    cwd,
    stdin: "inherit",
    stdout: "inherit",
    stderr: "inherit",
  }).output();
  if (code !== 0) {
    throw new Error(`${cmd} ${args.join(" ")} exited with ${code}`);
  }
}

async function output(
  cmd: string,
  args: string[],
  cwd: string,
): Promise<string> {
  const result = await new Deno.Command(cmd, {
    args,
    cwd,
    stdout: "piped",
    stderr: "inherit",
  }).output();
  if (!result.success) {
    throw new Error(`${cmd} ${args.join(" ")} exited with ${result.code}`);
  }
  return new TextDecoder().decode(result.stdout).trim();
}

export async function signRelease(
  options: SignReleaseOptions = {},
): Promise<string> {
  const root = options.root ?? Deno.cwd();
  // packageVersion drops the "v"; release tags keep it.
  const tag = options.tag ??
    "v" + packageVersion(
        await Deno.readTextFile(join(root, "cli", "meta", "version.go")),
      );
  const repo = ["--repo", REPO];

  // Uploading to a published release would change assets people already
  // downloaded, and winget.yml has already run against them.
  const isDraft = await output(
    "gh",
    ["release", "view", tag, ...repo, "--json", "isDraft", "--jq", ".isDraft"],
    root,
  );
  if (isDraft !== "true") {
    throw new Error(
      `${tag} is not a draft release; push the tag and wait for the Release workflow`,
    );
  }

  // Sign what CI published, not a local build, which can differ.
  await Deno.mkdir(join(root, "build"), { recursive: true });
  await run("gh", [
    "release",
    "download",
    tag,
    ...repo,
    "--pattern",
    "SHA256SUMS",
    "--dir",
    "build",
    "--clobber",
  ], root);

  const sums = join("build", "SHA256SUMS");
  await run("minisign", [
    "-S",
    "-m",
    sums,
    "-t",
    `ZVM ${tag} release checksums`,
  ], root);
  // Fails if the local secret key isn't the one users are told to trust.
  await run("minisign", ["-V", "-m", sums, "-p", "minisign.pub"], root);

  await run("gh", [
    "release",
    "upload",
    tag,
    ...repo,
    `${sums}.minisig`,
    "--clobber",
  ], root);
  return tag;
}

if (import.meta.main) {
  if (Deno.args.length > 1) {
    console.error("usage: deno task release:sign [tag]");
    Deno.exit(2);
  }
  try {
    const tag = await signRelease({ tag: Deno.args[0] });
    console.log(
      `Signed ${tag}. Review the draft and publish it: https://github.com/${REPO}/releases`,
    );
  } catch (err) {
    console.error(err instanceof Error ? err.message : err);
    Deno.exit(1);
  }
}
