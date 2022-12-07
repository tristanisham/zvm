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

console.time("Built zvm")

for (const os of GOOS) {
    for (const ar of GOARCH) {
        if (os == "solaris" && ar == "arm64" || os == "plan9" && ar == "arm64") continue;
        Deno.env.set("GOOS", os)
        Deno.env.set("GOARCH", ar)
        console.time(`Build zvm: ${os}-${ar}`)
        await exec(`go build -o build/${os}-${ar}/zvm`)
        console.timeEnd(`Build zvm: ${os}-${ar}`)
    }
}

Deno.chdir("build")
for (const os of GOOS) {
    for (const ar of GOARCH) {
        if (os == "solaris" && ar == "arm64" || os == "plan9" && ar == "arm64") continue;
        console.time(`Compress zvm: ${os}-${ar}`)
        await exec(`tar -czf ${os}-${ar}.tar.gz ${os}-${ar}`)
        console.timeEnd(`Compress zvm: ${os}-${ar}`)
    }
}

console.timeEnd(`Built zvm`)