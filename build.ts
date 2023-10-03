#!/usr/bin/env -S deno run -A
import { Tar } from "https://deno.land/std@0.184.0/archive/mod.ts";
import { copy } from "https://deno.land/std@0.184.0/streams/copy.ts";

const GOARCH = [
  "amd64",
  "arm64",
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

for (const os of GOOS) {
  for (const ar of GOARCH) {
    if (os == "solaris" && ar == "arm64" || os == "plan9" && ar == "arm64") {
      continue;
    }
    Deno.env.set("GOOS", os);
    Deno.env.set("GOARCH", ar);
    const zvm_str = `zvm-${os}-${ar}`;
    console.time(`Build zvm: ${zvm_str}`);
    // deno-lint-ignore no-deprecated-deno-api
    const build_cmd = Deno.run({
      cmd: [
        "go",
        "build",
        "-o",
        `build/${zvm_str}/zvm${(os == "windows" ? ".exe" : "")}`,
        "-ldflags=-w -s", "-buildmode=pie", "-trimpath",
      ],
    });

    const { code } = await build_cmd.status();
    if (code !== 0) {
      console.error("Something went wrong");
      Deno.exit(1);
    }

    console.timeEnd(`Build zvm: ${zvm_str}`);
  }
}

Deno.chdir("build");
for (const os of GOOS) {
  for (const ar of GOARCH) {
    if (os == "solaris" && ar == "arm64" || os == "plan9" && ar == "arm64") {
      continue;
    }
    const zvm_str = `zvm-${os}-${ar}`;

    if (os == "windows") {
      console.time(`Compress zvm: ${zvm_str}`);
      const zip = new Deno.Command(`zip`, {
        args: [`${zvm_str}.zip`, `${zvm_str}/zvm.exe`],
        stdin: "piped",
        stdout: "piped",
      });
      zip.spawn();
      console.timeEnd(`Compress zvm: ${zvm_str}`);
      continue;
    }
    const tar = new Tar();
    console.time(`Compress zvm: ${zvm_str}`);
    await tar.append("zvm", {
      filePath: `${zvm_str}/zvm`,
    });
    const writer = await Deno.open(`./${zvm_str}.tar`, {
      write: true,
      create: true,
    });
    await copy(tar.getReader(), writer);
    writer.close();
    console.timeEnd(`Compress zvm: ${zvm_str}`);
  }
}

console.timeEnd(`Built zvm`);
