// Run this with Deno
// deno run -A build.ts
import { exec } from "https://deno.land/x/exec@0.0.5/mod.ts";

const OS = [
    "windows",
    "darwin",
    "linux"
];

const ARCH = [
    "arm64",
    "amd64"
];

const distNames: string[] = [];

for (const o of OS) {
    for (const a of ARCH) {
        Deno.env.set("GOOS", o)
        Deno.env.set("GOARCH", a)
        await exec(`go build -o build/zvm-${o}-${a}/zvm`)

        const distName = `zvm-${o}-${a}`
        distNames.push(distName);
    }
}

Deno.chdir("build")
for (const name of distNames) {
    await exec(`tar -czvf ${name}.tar.gz ${name}`)
}