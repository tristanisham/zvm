// deno-lint-ignore-file no-import-prefix
// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

import { Tar } from "https://deno.land/std@0.184.0/archive/mod.ts";
import { copy } from "https://deno.land/std@0.184.0/streams/copy.ts";
import { parseArgs } from "@std/cli/parse-args";

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
  if (BuildUpgradeMessage === "" || BuildUpgradeMessage === undefined) {
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

const projectRoot = Deno.cwd();

await Deno.mkdir("./build", { recursive: true });

console.time("Built zvm");
Deno.env.set("CGO_ENABLED", "0");

const targets = getTargets();

// Compile step — all targets in parallel
const compileResults = await Promise.all(
  targets.map(async ({ os, arch, label }) => {
    console.time(`Build zvm: ${label}`);

    const buildPath = `build/${label}/zvm${os === "windows" ? ".exe" : ""}`;

    const build_cmd = new Deno.Command("go", {
      args: [
        "build",
        ...(args.autoUpgrades ? [] : ["-tags", "noAutoUpgrades"]),
        "-o",
        buildPath,
        `-ldflags=-w -s -X 'main.BuildUpgradeMessage=${BuildUpgradeMessage}'`,
        "-trimpath",
      ],
      env: {
        ...Deno.env.toObject(),
        GOOS: os,
        GOARCH: arch,
      },
    });

    const { code, stderr } = await build_cmd.output();
    if (code !== 0) {
      console.error(`Failed to build ${label}:`);
      console.error(new TextDecoder().decode(stderr));
      Deno.exit(1);
    }

    console.timeEnd(`Build zvm: ${label}`);
    return `${projectRoot}/build/${label}`;
  }),
);

// Bundle step — all targets in parallel
await Promise.all(
  // deno-lint-ignore no-unused-vars
  targets.map(async ({ os, arch, label }) => {
    const buildDir = `${projectRoot}/build`;

    if (os === "windows") {
      console.time(`Compress zvm (zip): ${label}`);
      const zip = new Deno.Command("zip", {
        args: [
          `${label}.zip`,
          `${label}/zvm.exe`,
          `${label}/elevate.cmd`,
          `${label}/elevate.vbs`,
        ],
        cwd: buildDir,
      });

      const { code, stderr } = await zip.output();
      if (code !== 0) {
        console.error(`Failed to zip ${label}:`);
        console.error(new TextDecoder().decode(stderr));
      }
      console.timeEnd(`Compress zvm (zip): ${label}`);
      return;
    }

    console.time(`Compress zvm (tar): ${label}`);
    const tar = new Tar();
    await tar.append("zvm", {
      filePath: `${buildDir}/${label}/zvm`,
    });

    const writer = await Deno.open(`${buildDir}/${label}.tar`, {
      write: true,
      create: true,
    });
    await copy(tar.getReader(), writer);
    writer.close();
    console.timeEnd(`Compress zvm (tar): ${label}`);
  }),
);

console.timeEnd("Built zvm");

// Cleanup uncompressed directories
console.time("Remove build artifacts");
await Promise.all(
  compileResults.map((dir) => Deno.remove(dir, { recursive: true })),
);
console.timeEnd("Remove build artifacts");
