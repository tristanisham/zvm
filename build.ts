// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

import { parseArgs } from "@std/cli/parse-args";
import { TarStream, type TarStreamInput } from "@std/tar";
import * as zip from "@zip-js/zip-js";

// Command to count final build results
//  find ./build -type f \( -name "*.tar" -o -name "*.zip" \) | wc -l

const args = parseArgs(Deno.args, {
  string: ["buildUpgradeMessage"],
  boolean: ["autoUpgrades"],
  negatable: ["autoUpgrades"],
  default: { autoUpgrades: true },
});

const BuildUpgradeMessage = args.buildUpgradeMessage || "";

if (!args.autoUpgrades) {
  console.log(
    "%cBuilding without autoUpgrades (noAutoUpgrades)",
    "color: yellow;",
  );
  if (BuildUpgradeMessage === "") {
    console.warn(
      "%cbuildUpgradeMessage not set, falling back to default message",
      "color: red;",
    );
  }
}

const GOARCH = [
  "amd64",
  "arm64",
  "loong64",
  "ppc64le",
];

const GOOS = [
  "windows",
  "linux",
  "darwin",
  "freebsd",
  "netbsd",
  "openbsd",
  // "plan9",
  "solaris",
];

interface Target {
  os: string;
  arch: string;
  label: string;
}

function getTargets(): Target[] {
  const targets: Target[] = [];
  for (const os of GOOS) {
    for (const arch of GOARCH) {
      if (
        os === "solaris" && arch === "arm64" ||
        os === "plan9" && arch === "arm64" ||
        os !== "linux" && arch === "loong64" ||
        os !== "linux" && arch === "ppc64le"
      ) {
        continue;
      }
      targets.push({ os, arch, label: `zvm-${os}-${arch}` });
    }
  }
  return targets;
}

const buildDir = `${Deno.cwd()}/build`;
await Deno.mkdir(buildDir, { recursive: true });

// Snapshot the environment once; per-target GOOS/GOARCH are layered on top.
const baseEnv = { ...Deno.env.toObject(), CGO_ENABLED: "0" };

// Build, archive, and clean up a single target. Pipelining the three steps
// per target lets compression of finished builds overlap with in-flight
// compiles, and removing each directory as soon as it's archived keeps peak
// disk usage to one uncompressed binary per worker.
async function buildTarget({ os, arch, label }: Target): Promise<void> {
  const outDir = `${buildDir}/${label}`;
  const binName = `zvm${os === "windows" ? ".exe" : ""}`;
  const binPath = `${outDir}/${binName}`;

  console.time(`Build zvm: ${label}`);
  const { code, stderr } = await new Deno.Command("go", {
    args: [
      "build",
      ...(args.autoUpgrades ? [] : ["-tags", "noAutoUpgrades"]),
      "-o",
      binPath,
      `-ldflags=-w -s -X 'main.BuildUpgradeMessage=${BuildUpgradeMessage}'`,
      "-trimpath",
    ],
    env: { ...baseEnv, GOOS: os, GOARCH: arch },
  }).output();
  if (code !== 0) {
    throw new Error(
      `Failed to build ${label}:\n${new TextDecoder().decode(stderr)}`,
    );
  }
  console.timeEnd(`Build zvm: ${label}`);

  console.time(`Compress zvm: ${label}`);
  if (os === "windows") {
    await zipFile(binPath, binName, `${buildDir}/${label}.zip`);
  } else {
    await tarFile(binPath, binName, `${buildDir}/${label}.tar`);
  }
  console.timeEnd(`Compress zvm: ${label}`);

  await Deno.remove(outDir, { recursive: true });
}

async function tarFile(
  src: string,
  entryName: string,
  dest: string,
): Promise<void> {
  const file = await Deno.open(src);
  const { size } = await file.stat();
  await ReadableStream.from<TarStreamInput>([
    { type: "file", path: entryName, size, readable: file.readable },
  ])
    .pipeThrough(new TarStream())
    .pipeTo((await Deno.create(dest)).writable);
}

async function zipFile(
  src: string,
  entryName: string,
  dest: string,
): Promise<void> {
  const writer = new zip.ZipWriter(new zip.BlobWriter("application/zip"));
  const file = await Deno.open(src);
  await writer.add(entryName, file.readable);
  const blob = await writer.close();
  await blob.stream().pipeTo((await Deno.create(dest)).writable);
}

const targets = getTargets();

// Each `go build` parallelizes internally, so cap concurrent targets at the
// CPU count instead of launching all of them at once.
const concurrency = Math.min(
  Math.max(1, navigator.hardwareConcurrency ?? 4),
  targets.length,
);

console.time("Built zvm");

const queue = [...targets];
const failures: string[] = [];
await Promise.all(
  Array.from({ length: concurrency }, async () => {
    for (let t = queue.shift(); t !== undefined; t = queue.shift()) {
      try {
        await buildTarget(t);
      } catch (err) {
        failures.push(err instanceof Error ? err.message : String(err));
      }
    }
  }),
);

console.timeEnd("Built zvm");

if (failures.length > 0) {
  for (const failure of failures) {
    console.error(`%c${failure}`, "color: red;");
  }
  Deno.exit(1);
}
