// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

import { dirname, join } from "node:path";
import { packageVersionFiles, syncPackageVersions } from "./pkg.ts";

function assertEquals(actual: unknown, expected: unknown): void {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(
      `Values differ:\nactual: ${JSON.stringify(actual)}\nexpected: ${
        JSON.stringify(expected)
      }`,
    );
  }
}

async function fixture(): Promise<string> {
  const root = await Deno.makeTempDir();
  const files = new Map<string, string>([
    [
      join("cli", "meta", "version.go"),
      'package meta\n\nconst (\n\tVERSION = "v1.2.3"\n)\n',
    ],
    [
      "flake.nix",
      '{ packages.default = buildGoModule { pname = "zvm"; version = "0.1.0"; }; }\n',
    ],
    [
      join(".claude-plugin", "plugin.json"),
      '{\n  "name": "zvm",\n  "version": "0.1.0"\n}\n',
    ],
    [
      join(".claude-plugin", "marketplace.json"),
      '{\n  "plugins": [{ "name": "zvm", "version": "0.1.0" }]\n}\n',
    ],
    [
      join(".codex-plugin", "plugin.json"),
      '{\n  "name": "zvm",\n  "version": "0.1.0"\n}\n',
    ],
  ]);

  for (const [path, contents] of files) {
    const destination = join(root, path);
    await Deno.mkdir(dirname(destination), { recursive: true });
    await Deno.writeTextFile(destination, contents);
  }

  return root;
}

Deno.test("syncPackageVersions updates every distribution version", async () => {
  const root = await fixture();
  try {
    const result = await syncPackageVersions({ root });
    assertEquals(result.version, "1.2.3");
    assertEquals([...result.updated].sort(), [...packageVersionFiles].sort());

    for (const path of packageVersionFiles) {
      const contents = await Deno.readTextFile(join(root, path));
      if (!contents.includes('"1.2.3"')) {
        throw new Error(`${path} was not synchronized: ${contents}`);
      }
    }

    const secondRun = await syncPackageVersions({ root });
    assertEquals(secondRun.updated, []);
  } finally {
    await Deno.remove(root, { recursive: true });
  }
});

Deno.test("syncPackageVersions check reports drift without writing", async () => {
  const root = await fixture();
  try {
    let message = "";
    try {
      await syncPackageVersions({ root, check: true });
    } catch (error) {
      message = error instanceof Error ? error.message : String(error);
    }

    if (!message.includes("deno task pkg")) {
      throw new Error(`check did not report how to fix drift: ${message}`);
    }

    const flake = await Deno.readTextFile(join(root, "flake.nix"));
    if (!flake.includes('version = "0.1.0"')) {
      throw new Error("check mode modified flake.nix");
    }
  } finally {
    await Deno.remove(root, { recursive: true });
  }
});
