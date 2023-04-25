import { exec } from "https://deno.land/x/exec@0.0.5/mod.ts";
import * as zip from "https://deno.land/x/zipjs@v2.7.6/index.js";
import { Tar } from "https://deno.land/std@0.184.0/archive/mod.ts";
import { copy } from "https://deno.land/std@0.184.0/streams/copy.ts";




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
        // Deno.env.set("CGO_ENABLED", "1")
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

        if (os == "windows") {
            console.time(`Compress zvm: ${zvm_str}`)
            const zipWriter = new zip.BlobWriter();
            const output = new zip.TextReader(await Deno.readTextFile(`${zvm_str}/zvm`))
            const zipper = new zip.ZipWriter(zipWriter);
            await zipper.add("zvm", output)
            console.timeEnd(`Compress zvm: ${zvm_str}`)
            continue;
        }

        const tar = new Tar()
        console.time(`Compress zvm: ${zvm_str}`)
        await tar.append("zvm", {
            filePath: `${zvm_str}/zvm`
        })
        const writer = await Deno.open(`./${zvm_str}.tar`, { write: true, create: true })
        await copy(tar.getReader(), writer);
        writer.close();
        console.timeEnd(`Compress zvm: ${zvm_str}`)

    }
}

console.timeEnd(`Built zvm`)

