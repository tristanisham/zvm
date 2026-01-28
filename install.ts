#!/usr/bin/env -S deno run --allow-net=github.com -W

import { parseFlags } from "@cliffy/flags";
import ProgressBar from "@deno-library/progress";

if (import.meta.main) {
  const { flags } = parseFlags(Deno.args, {
    stopEarly: true,
    flags: [{
      name: "os",
      type: "string",
      default: Deno.build.os,
    }, {
      name: "arch",
      type: "string",
      default: Deno.build.arch,
    }],
  });

  let arch: string = flags.arch;
  const os = flags.os;
  const ext = os === "windows" ? "zip" : "tar";

  switch (arch) {
    case "x86_64":
      arch = "amd64";
      break;
    case "aarch64":
      arch = "arm64";
      break;
  }

  switch (os) {
    case "darwin":
    case "linux":
    // case "android":
    case "windows":
    case "freebsd":
    case "netbsd":
    // case "aix":
    // case "illumos":
    case "solaris":
      break;
    default: {
      console.warn("Your operating system is not supported by ZVM");
      console.log(
        "If you would like to manually pass in a OS, please use the %c--os%c flag",
        "color: yellow",
      );
      const platforms = [
        "darwin",
        "linux",
        "android",
        "windows",
        "freebsd",
        "netbsd",
        "aix",
        "solaris",
        "illumos",
      ];
      platforms.forEach((p) => console.log(`\t${p}`));
    }
  }

  const url =
    `https://github.com/tristanisham/zvm/releases/download/latest/zvm-${os}-${arch}.${ext}`;
  const temp = await Deno.makeTempFile({
    prefix: `zvm-${os}-${arch}`,
    suffix: ext,
  });
  const bundle = await Deno.open(temp, { write: true, create: true });

  const resp = await fetch(url);
  const totalStr = resp.headers.get("content-length");
  const total = totalStr ? parseInt(totalStr, 10) : 0;

  if (!resp.body) throw new Error("Response body is null");

  const progress = new ProgressBar({
    total,
    display: ":bar :percent :eta :completed/:total",
  });

  try {
    const reader = resp.body?.getReader();
    if (!reader) throw new Error("Response body is not readable");
    let downloaded = 0;

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      await bundle.write(value);
      downloaded += value.byteLength;
      progress.render(downloaded);
    }

    console.log(`\nZVM bundle saved to {tmp}`);
  } finally {
    bundle.close();
  }
}
