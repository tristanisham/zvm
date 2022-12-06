import { exec } from "https://deno.land/x/exec@0.0.5/mod.ts";
const GOARCH = [
    "amd64", "arm64"
];

const GOOS = [
    "windows", "linux", "darwin", "freebsd", "netbsd",
    "openbsd",
    "plan9",
    "solaris",
]

for (const os of GOOS) {
    for (const ar of GOARCH) {
        Deno.env.set("GOOS", os)
        Deno.env.set("GOARCH", ar)
        await exec(`go build -o build/${os}-${ar}/zvm`)
    }
}

Deno.chdir("build")
for (const os of GOOS) {
    for (const ar of GOARCH) {
        await exec(`tar -czvf ${os}-${ar}.tar.gz ${os}-${ar}`)
    }
}