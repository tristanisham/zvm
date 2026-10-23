// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

import { fetchStarTimestamps, githubToken } from "./star-history.ts";

async function withEnv(
  env: Record<string, string | undefined>,
  run: () => Promise<void>,
): Promise<void> {
  const previous = Object.fromEntries(
    Object.keys(env).map((name) => [name, Deno.env.get(name)]),
  );
  try {
    for (const [name, value] of Object.entries(env)) {
      if (value === undefined) Deno.env.delete(name);
      else Deno.env.set(name, value);
    }
    await run();
  } finally {
    for (const [name, value] of Object.entries(previous)) {
      if (value === undefined) Deno.env.delete(name);
      else Deno.env.set(name, value);
    }
  }
}

Deno.test("star history prefers GH_TOKEN and accepts GITHUB_TOKEN", async () => {
  for (const gh of ["gh-fixture", undefined]) {
    await withEnv(
      { GH_TOKEN: gh, GITHUB_TOKEN: "actions-fixture" },
      async () => {
        if (await githubToken() !== (gh ?? "actions-fixture")) {
          throw new Error("wrong token precedence");
        }
      },
    );
  }
});

Deno.test({
  name: "star history uses the saved gh login without environment tokens",
  ignore: Deno.build.os === "windows",
  async fn() {
    const dir = await Deno.makeTempDir();
    try {
      await Deno.writeTextFile(
        `${dir}/gh`,
        '#!/bin/sh\n[ "$*" = "auth token --hostname github.com" ] || exit 1\nprintf "saved-login-fixture\\n"\n',
      );
      await Deno.chmod(`${dir}/gh`, 0o755);
      await withEnv(
        { GH_TOKEN: undefined, GITHUB_TOKEN: undefined, PATH: dir },
        async () => {
          if (await githubToken() !== "saved-login-fixture") {
            throw new Error("saved gh login was not used");
          }
        },
      );
    } finally {
      await Deno.remove(dir, { recursive: true });
    }
  },
});

Deno.test("star history permits anonymous access when gh is unavailable", async () => {
  await withEnv(
    { GH_TOKEN: undefined, GITHUB_TOKEN: undefined, PATH: "" },
    async () => {
      if (await githubToken() !== undefined) {
        throw new Error("unexpected token without credentials");
      }
    },
  );
});

Deno.test("star history authenticates requests and explains rejected credentials", async () => {
  const originalFetch = globalThis.fetch;
  try {
    await withEnv({ GH_TOKEN: "fixture-token" }, async () => {
      for (const status of [200, 401]) {
        globalThis.fetch = (_input, init) => {
          if (
            new Headers(init?.headers).get("Authorization") !==
              "Bearer fixture-token"
          ) {
            throw new Error("request omitted authentication");
          }
          return Promise.resolve(
            Response.json([{ starred_at: "2026-01-01T00:00:00Z" }], { status }),
          );
        };
        if (status === 200) {
          const dates = await fetchStarTimestamps("owner/repo");
          if (dates.length !== 1) throw new Error("missing star timestamp");
        } else {
          let message = "";
          try {
            await fetchStarTimestamps("owner/repo");
          } catch (err) {
            message = String(err);
          }
          if (!message.includes("401") || !message.includes("gh auth login")) {
            throw new Error(
              "authentication error omitted recovery instructions",
            );
          }
        }
      }
    });
  } finally {
    globalThis.fetch = originalFetch;
  }
});
