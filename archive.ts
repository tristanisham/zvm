import * as path from "https://deno.land/std@0.203.0/path/mod.ts"
import * as color from "https://deno.land/std@0.203.0/fmt/colors.ts"

if (import.meta.main) {
    await dailyArchive();
}

async function dailyArchive() {
    const URL = "https://ziglang.org/download/index.json";
    const resp = await fetch(URL);
    const body: ZigVersion = await resp.json();
    const master = body.master;
    const todayPath = `archive/${master.date}`
    Deno.mkdir(todayPath, { recursive: true });

    const targets = [
        master["x86_64-macos"],
        master["aarch64-macos"],
        master["x86_64-linux"],
        master["aarch64-linux"],
        master["armv7a-linux"],
        master["riscv64-linux"],
        master["powerpc64le-linux"],
        master["powerpc-linux"],
        master["x86-linux"],
        master["x86_64-windows"],
        master["aarch64-windows"],
        master["x86-windows"],
    ];

    for (const entry of targets) {
        try {
            const download = await fetch(entry.tarball);
            if (!download.ok) {
                console.error('Failed to download', entry.tarball, ':', download.statusText);
                continue;
            }

            const fileName = path.basename(entry.tarball);
            console.log(`Fetching ${color.green(fileName)}`)
            const file = await Deno.open(path.join(todayPath, fileName), { create: true, write: true });
            if (download.body) {
                await download.body.pipeTo(file.writable);
            } else {
                console.error('No body in response for', entry.tarball);
            }
            // file.close();
        } catch (error) {
            console.error('Error processing', entry.tarball, ':', error);
        }
    }
}




export interface ZigVersion {
    master: Master
}

export interface Master {
    version: string
    date: string
    docs: string
    stdDocs: string
    src: Info
    bootstrap: Info
    "x86_64-macos": Info
    "aarch64-macos": Info
    "x86_64-linux": Info
    "aarch64-linux": Info
    "armv7a-linux": Info
    "riscv64-linux": Info
    "powerpc64le-linux": Info
    "powerpc-linux": Info
    "x86-linux": Info
    "x86_64-windows": Info
    "aarch64-windows": Info
    "x86-windows": Info
}

export interface Info {
    tarball: string
    shasum: string
    size: string
}

