// faster_build.ts
// Copyright 2025 Tristan Isham.
// MIT license (see LICENSE).

// Run with: deno run -A faster_build.ts
// Optional: JOBS=6 deno run -A faster_build.ts

const GOARCH = ["amd64", "arm64", "loong64", "ppc64le"] as const;
const GOOS = ["windows", "linux", "darwin", "freebsd", "netbsd", "openbsd", "solaris"] as const;

type OS = (typeof GOOS)[number];
type Arch = (typeof GOARCH)[number];

const validCombo = (os: OS, ar: Arch) =>
  !(
    (os === "solaris" && ar === "arm64") ||
    (os !== "linux" && (ar === "loong64" || ar === "ppc64le"))
  );

const combos = GOOS.flatMap((os) => GOARCH.map((ar) => ({ os, ar } as const)))
  .filter(({ os, ar }) => validCombo(os, ar));

const JOBS = Number(Deno.env.get("JOBS") ?? Math.max(2, Math.min(6, Deno.systemMemoryInfo?.() ? 6 : 4)));
const BUILD_DIR = "build";

await Deno.mkdir(BUILD_DIR, { recursive: true });

console.time("Built zvm");

// ---- helpers ---------------------------------------------------------------

async function commandExists(cmd: string): Promise<boolean> {
  try {
    const p = new Deno.Command(cmd, { args: ["--version"] });
    const { code } = await p.output();
    return code === 0;
  } catch {
    return false;
  }
}

// Use external `zip` (with cwd=BUILD_DIR) — archive path must NOT repeat BUILD_DIR.
async function zipWithCLI(zvmStr: string): Promise<void> {
  const zip = new Deno.Command("zip", {
    args: [
      "-q",
      `${zvmStr}.zip`,
      `${zvmStr}/zvm.exe`,
    ],
    cwd: BUILD_DIR,
  });
  const { code, stdout, stderr } = await zip.output();
  if (code !== 0) {
    // Surface both streams; some `zip` builds log to stdout.
    const td = new TextDecoder();
    console.error(td.decode(stdout));
    console.error(td.decode(stderr));
    throw new Error(`zip failed (cli): ${zvmStr}`);
  }
}

// Fallback: use jsr:@zip-js/zip-js to create the zip in-memory then write to disk.
async function zipWithJSR(zvmStr: string): Promise<void> {
  // Dynamic import only when needed
  const { ZipWriter, Uint8ArrayReader, BlobWriter } = await import("jsr:@zip-js/zip-js");

  const exePath = `${BUILD_DIR}/${zvmStr}/zvm.exe`;
  const bytes = await Deno.readFile(exePath);

  const writer = new ZipWriter(new BlobWriter("application/zip"));
  // Keep same internal layout as CLI zip: <folder>/zvm.exe
  await writer.add(`${zvmStr}/zvm.exe`, new Uint8ArrayReader(bytes));
  const blob = await writer.close();

  const out = new Uint8Array(await blob.arrayBuffer());
  await Deno.writeFile(`${BUILD_DIR}/${zvmStr}.zip`, out);
}

// ---- build prep ------------------------------------------------------------

// One-time module download to warm the cache
{
  const prep = new Deno.Command("go", { args: ["mod", "download"] });
  const { code, stderr } = await prep.output();
  if (code !== 0) {
    console.error(new TextDecoder().decode(stderr));
    Deno.exit(1);
  }
}

async function buildOne(os: OS, ar: Arch) {
  const zvmStr = `zvm-${os}-${ar}`;
  const outDir = `${BUILD_DIR}/${zvmStr}`;
  await Deno.mkdir(outDir, { recursive: true });

  console.time(`Build ${zvmStr}`);
  const build = new Deno.Command("go", {
    args: [
      "build",
      "-o",
      `${outDir}/zvm${os === "windows" ? ".exe" : ""}`,
      "-ldflags=-w -s",
      "-trimpath",
      "-buildvcs=false",
      "-p", String(Math.max(2, Math.min(8, JOBS))), // tune per machine
    ],
    env: {
      GOOS: os,
      GOARCH: ar,
      CGO_ENABLED: "0",
    },
  });

  const { code, stdout, stderr } = await build.output();
  console.timeEnd(`Build ${zvmStr}`);
  if (code !== 0) {
    const td = new TextDecoder();
    console.error(td.decode(stdout));
    console.error(td.decode(stderr));
    throw new Error(`build failed: ${zvmStr}`);
  }

  // Bundle
  if (os === "windows") {
    console.time(`Zip ${zvmStr}`);
    const hasZip = await commandExists("zip");
    try {
      if (hasZip) {
        await zipWithCLI(zvmStr);
      } else {
        await zipWithJSR(zvmStr);
      }
    } finally {
      console.timeEnd(`Zip ${zvmStr}`);
    }
  } else {
    console.time(`Tar ${zvmStr}`);
    // Requires `tar` in PATH. Plain .tar (no compression).
    const tar = new Deno.Command("tar", {
      args: ["-cf", `${zvmStr}.tar`, `${zvmStr}/zvm`],
      cwd: BUILD_DIR,
    });
    const { code: tc, stdout: tOut, stderr: tErr } = await tar.output();
    console.timeEnd(`Tar ${zvmStr}`);
    if (tc !== 0) {
      const td = new TextDecoder();
      console.error(td.decode(tOut));
      console.error(td.decode(tErr));
      throw new Error(`tar failed: ${zvmStr}`);
    }
  }
}

async function runPool<T>(items: T[], worker: (t: T) => Promise<void>, concurrency: number) {
  const q = items.slice();
  const workers = Array.from({ length: concurrency }, async () => {
    while (q.length) {
      const next = q.shift();
      if (!next) break;
      try {
        await worker(next);
      } catch (e) {
        // Fail fast; surface first error
        throw e;
      }
    }
  });
  await Promise.all(workers);
}

// Build + bundle with a small worker pool
await runPool(combos, ({ os, ar }) => buildOne(os, ar), JOBS);

console.timeEnd("Built zvm");

// Tip: to count final artifacts
//   find ./build -type f \( -name "*.tar" -o -name "*.zip" \) | wc -l
