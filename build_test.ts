// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { packageVersionFiles } from "./.github/scripts/pkg.ts";

for (const fail of [false, true]) {
  Deno.test({
    name: `build preserves legacy binaries and cleans workspaces on ${
      fail ? "failure" : "success"
    }`,
    ignore: Deno.build.os === "windows",
    async fn() {
      const root = await Deno.makeTempDir();
      try {
        for (const path of ["cli/meta/version.go", ...packageVersionFiles]) {
          const dest = join(root, path);
          await Deno.mkdir(dirname(dest), { recursive: true });
          await Deno.copyFile(new URL(path, import.meta.url), dest);
        }
        const buildDir = join(root, "build");
        await Deno.mkdir(buildDir);
        const legacy = join(buildDir, "zvm-linux-386");
        await Deno.writeTextFile(legacy, "legacy binary");

        const toolsDir = join(root, "tools");
        await Deno.mkdir(toolsDir);
        const fakeGo = join(toolsDir, "go");
        await Deno.writeTextFile(
          fakeGo,
          `#!/bin/sh
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-o" ]; then
    shift
    printf 'test binary' > "$1" || exit 1
    exit ${fail ? 1 : 0}
  fi
  shift
done
exit 2
`,
        );
        await Deno.chmod(fakeGo, 0o755);
        const result = await new Deno.Command(Deno.execPath(), {
          args: [
            "run",
            "-A",
            "--cached-only",
            "--config",
            fileURLToPath(new URL("deno.jsonc", import.meta.url)),
            fileURLToPath(new URL("build.ts", import.meta.url)),
          ],
          cwd: root,
          env: { PATH: toolsDir },
          stdout: "piped",
          stderr: "piped",
        }).output();
        if (result.code !== (fail ? 1 : 0)) {
          throw new Error(new TextDecoder().decode(result.stderr));
        }
        if (await Deno.readTextFile(legacy) !== "legacy binary") {
          throw new Error("build overwrote a legacy binary");
        }
        const entries = Array.from(Deno.readDirSync(buildDir));
        if (entries.some((entry) => entry.isDirectory)) {
          throw new Error("build left temporary workspaces behind");
        }
        if (!fail) {
          for (
            const archive of ["zvm-linux-386.tar", "zvm-windows-amd64.zip"]
          ) {
            const info = await Deno.stat(join(buildDir, archive));
            if (info.size === 0) throw new Error(`empty archive: ${archive}`);
          }
        }
      } finally {
        await Deno.remove(root, { recursive: true });
      }
    },
  });
}
