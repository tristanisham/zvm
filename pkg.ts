// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

import { join } from "node:path";

const versionSource = join("cli", "meta", "version.go");

interface PackageVersionTarget {
  path: string;
  pattern: RegExp;
}

export const packageVersionFiles = [
  "flake.nix",
  join(".claude-plugin", "plugin.json"),
  join(".claude-plugin", "marketplace.json"),
  join(".codex-plugin", "plugin.json"),
] as const;

const targets: PackageVersionTarget[] = [
  {
    path: packageVersionFiles[0],
    pattern: /(pname\s*=\s*"zvm";[\s\S]*?\bversion\s*=\s*")[^"]*(";)/,
  },
  ...packageVersionFiles.slice(1).map((path) => ({
    path,
    pattern: /("version"\s*:\s*")[^"]*(")/,
  })),
];

export interface SyncPackageVersionOptions {
  root?: string;
  check?: boolean;
}

export interface SyncPackageVersionResult {
  version: string;
  updated: string[];
}

function packageVersion(contents: string): string {
  const matches = [
    ...contents.matchAll(/^\s*VERSION\s*=\s*"([^"]+)"/gm),
  ];
  if (matches.length !== 1) {
    throw new Error(
      `${versionSource} must declare exactly one VERSION constant`,
    );
  }

  const version = matches[0][1];
  if (
    !/^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/.test(version)
  ) {
    throw new Error(
      `${versionSource} contains invalid release version ${version}`,
    );
  }

  return version.slice(1);
}

function replaceVersion(
  path: string,
  contents: string,
  pattern: RegExp,
  version: string,
): string {
  const flags = pattern.flags.includes("g")
    ? pattern.flags
    : pattern.flags + "g";
  const matches = [...contents.matchAll(new RegExp(pattern.source, flags))];
  if (matches.length !== 1) {
    throw new Error(
      `${path} must contain exactly one package version, found ${matches.length}`,
    );
  }

  return contents.replace(pattern, `$1${version}$2`);
}

export async function syncPackageVersions(
  options: SyncPackageVersionOptions = {},
): Promise<SyncPackageVersionResult> {
  const root = options.root ?? Deno.cwd();
  const source = await Deno.readTextFile(join(root, versionSource));
  const version = packageVersion(source);
  const pending: Array<{ path: string; contents: string }> = [];

  for (const target of targets) {
    const path = join(root, target.path);
    const contents = await Deno.readTextFile(path);
    const updated = replaceVersion(
      target.path,
      contents,
      target.pattern,
      version,
    );
    if (updated !== contents) {
      pending.push({ path: target.path, contents: updated });
    }
  }

  if (options.check && pending.length > 0) {
    throw new Error(
      `Package versions do not match ${versionSource} (${version}): ${
        pending.map(({ path }) => path).join(", ")
      }. Run \`deno task pkg\` to synchronize them.`,
    );
  }

  for (const update of pending) {
    await Deno.writeTextFile(join(root, update.path), update.contents);
  }

  return {
    version,
    updated: pending.map(({ path }) => path),
  };
}

if (import.meta.main) {
  const unknownArgs = Deno.args.filter((arg) => arg !== "--check");
  if (unknownArgs.length > 0) {
    console.error(`Unknown argument: ${unknownArgs[0]}`);
    Deno.exit(2);
  }

  try {
    const check = Deno.args.includes("--check");
    const result = await syncPackageVersions({ check });
    if (check || result.updated.length === 0) {
      console.log(`Package versions match ${versionSource}: ${result.version}`);
    } else {
      console.log(
        `Synced ${result.updated.join(", ")} to ${result.version}`,
      );
    }
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    Deno.exit(1);
  }
}
