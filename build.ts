// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

import { Tar } from "https://deno.land/std@0.184.0/archive/mod.ts";
import { copy } from "https://deno.land/std@0.184.0/streams/copy.ts";
// Command to count final build results
//  find ./build -type f \( -name "*.tar" -o -name "*.zip" \) | wc -l

const git = new Deno.Command("git", { args: ['rev-parse', '--abbrev-ref', 'HEAD'], stdout: "piped" });
const { code, stderr, stdout } = await git.output();
if (code !== 0) {
  console.error(new TextDecoder().decode(stderr));
  Deno.exit(1);
}

const branchName = new TextDecoder().decode(stdout);
if (branchName !== "master") {
  console.error("Watch out! You're not building on branch: %cmaster", "color:yellow;")
  Deno.exit(1);
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
  "plan9",
  "solaris",
];

await Deno.mkdir("./build", { recursive: true });

console.time("Built zvm");
Deno.env.set("CGO_ENABLED", "0");

// Compile step
for (const os of GOOS) {
  for (const ar of GOARCH) {
    if (
      os == "solaris" && ar == "arm64" ||
      os == "plan9" && ar == "arm64" ||
      os != "linux" && ar == "loong64" ||
      os != "linux" && ar == "ppc64le"
    ) {
      continue;
    }
    Deno.env.set("GOOS", os);
    Deno.env.set("GOARCH", ar);
    const zvm_str = `zvm-${os}-${ar}`;
    console.time(`Build zvm: ${zvm_str}`);

    const build_cmd = new Deno.Command("go", {
      args: [
        "build",
        "-o",
        `build/${zvm_str}/zvm${os === "windows" ? ".exe" : ""}`,
        "-ldflags=-w -s",
        "-trimpath",
      ],
    });

    const { code, stderr } = await build_cmd.output();
    if (code !== 0) {
      console.error(new TextDecoder().decode(stderr));
      Deno.exit(1);
    }

    //    if (os == "windows") {
    //      await Deno.mkdir(zvm_str, { recursive: true });
    //
    //   }

    console.timeEnd(`Build zvm: ${zvm_str}`);
  }
}

// Bundle step
Deno.chdir("build");
for (const os of GOOS) {
  for (const ar of GOARCH) {
    if (
      os == "solaris" && ar == "arm64" ||
      os == "plan9" && ar == "arm64" ||
      os != "linux" && ar == "loong64" ||
      os != "linux" && ar == "ppc64le"
    ) {
      continue;
    }
    const zvm_str = `zvm-${os}-${ar}`;

    /**
     * Windows
     */
    if (os === "windows") {
      console.time(`Compress zvm (zip): ${zvm_str}`);
      const zip = new Deno.Command(`zip`, {
        args: [
          `${zvm_str}.zip`,
          `${zvm_str}/zvm.exe`,
          `${zvm_str}/elevate.cmd`,
          `${zvm_str}/elevate.vbs`,
        ],
        stdin: "piped",
        stdout: "piped",
      });

      zip.spawn();

      console.timeEnd(`Compress zvm (zip): ${zvm_str}`);
      continue;
    }

    const tar = new Tar();
    console.time(`Compress zvm (tar): ${zvm_str}`);
    await tar.append("zvm", {
      filePath: `${zvm_str}/zvm`,
    });

    const writer = await Deno.open(`./${zvm_str}.tar`, {
      write: true,
      create: true,
    });
    await copy(tar.getReader(), writer);
    writer.close();
    console.timeEnd(`Compress zvm (tar): ${zvm_str}`);
  }
}

console.timeEnd(`Built zvm`);
