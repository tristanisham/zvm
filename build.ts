import { exec } from "https://deno.land/x/exec@0.0.5/mod.ts";
// import { parse } from "https://deno.land/std@0.167.0/flags/mod.ts";

// const args = parse(Deno.args)

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
        const zvm_str = `zvm-${os}-${ar}`
        console.time(`Build zvm: ${zvm_str}`)
        await exec(`go build -o build/${zvm_str}/zvm`)
        console.timeEnd(`Build zvm: ${zvm_str}`)
    }
}

Deno.chdir("build")
for (const os of GOOS) {
    for (const ar of GOARCH) {
        if (os == "solaris" && ar == "arm64" || os == "plan9" && ar == "arm64") continue;
        const zvm_str = `zvm-${os}-${ar}`

        console.time(`Compress zvm: ${zvm_str}`)
        await exec(`tar -czf ${zvm_str}.tar.gz ${zvm_str}`)
        console.timeEnd(`Compress zvm: ${zvm_str}`)
    }
}

console.timeEnd(`Built zvm`)

