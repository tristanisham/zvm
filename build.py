# Copyright 2025 Tristan Isham. All rights reserved.
# Use of this source code is governed by the MIT
# license that can be found in the LICENSE file.

import argparse
import os
import shutil
import subprocess
import sys
import tarfile
import time
import zipfile
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from pathlib import Path

# Command to count final build results
#  find ./build -type f \( -name "*.tar" -o -name "*.zip" \) | wc -l


@dataclass(frozen=True)
class Target:
    os: str
    arch: str
    label: str


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        prog="buildZVM", description="build script for zvm"
    )
    parser.add_argument("--buildUpgradeMessage", default="")
    parser.add_argument(
        "--autoUpgrades",
        dest="autoUpgrades",
        action=argparse.BooleanOptionalAction,
        default=True,
    )
    return parser.parse_args()


args = parse_args()
BuildUpgradeMessage = args.buildUpgradeMessage or ""

if not args.autoUpgrades:
    print("Building without autoUpgrades (noAutoUpgrades)")
    if BuildUpgradeMessage == "":
        print(
            "buildUpgradeMessage not set, falling back to default message",
            file=sys.stderr,
        )


GOARCH = [
    "amd64",
    "arm64",
    "loong64",
    "ppc64le",
]

GOOS = [
    "windows",
    "linux",
    "darwin",
    "freebsd",
    "netbsd",
    "openbsd",
    # "plan9",
    "solaris",
]


def get_targets() -> list[Target]:
    targets: list[Target] = []
    for goos in GOOS:
        for goarch in GOARCH:
            if (
                goos == "solaris"
                and goarch == "arm64"
                or goos == "plan9"
                and goarch == "arm64"
                or goos != "linux"
                and goarch == "loong64"
                or goos != "linux"
                and goarch == "ppc64le"
            ):
                continue
            targets.append(Target(os=goos, arch=goarch, label=f"zvm-{goos}-{goarch}"))
    return targets


build_dir = Path.cwd() / "build"
build_dir.mkdir(parents=True, exist_ok=True)

# Snapshot the environment once; per-target GOOS/GOARCH are layered on top.
base_env = {**os.environ, "CGO_ENABLED": "0"}

ORANGE = "\033[38;5;208m"
AZUL = "\033[34m"
RESET = "\033[0m"


def format_elapsed(seconds: float) -> str:
    milliseconds = seconds * 1000
    if milliseconds < 1000:
        return f"{milliseconds:.3f}ms"
    return f"{seconds:.3f}s"


# Build, archive, and clean up a single target. Pipelining the three steps
# per target lets compression of finished builds overlap with in-flight
# compiles, and removing each directory as soon as it's archived keeps peak
# disk usage to one uncompressed binary per worker.
def build_target(target: Target) -> None:
    out_dir = build_dir / target.label
    bin_name = f"zvm{'.exe' if target.os == 'windows' else ''}"
    bin_path = out_dir / bin_name

    build_label = f"{ORANGE}Build{RESET} zvm: {target.label}"
    start = time.perf_counter()
    command = [
        "go",
        "build",
        *([] if args.autoUpgrades else ["-tags", "noAutoUpgrades"]),
        "-o",
        str(bin_path),
        "-ldflags=-w -s -X 'main.BuildUpgradeMessage=" + BuildUpgradeMessage + "'",
        "-trimpath",
    ]
    env = {**base_env, "GOOS": target.os, "GOARCH": target.arch}
    result = subprocess.run(command, env=env, stderr=subprocess.PIPE)
    if result.returncode != 0:
        raise RuntimeError(
            f"Failed to build {target.label}:\n{result.stderr.decode(errors='replace')}"
        )
    print(f"{build_label}: {format_elapsed(time.perf_counter() - start)}")

    compress_label = f"{AZUL}Compress{RESET} zvm: {target.label}"
    start = time.perf_counter()
    if target.os == "windows":
        zip_file(bin_path, bin_name, build_dir / f"{target.label}.zip")
    else:
        tar_file(bin_path, bin_name, build_dir / f"{target.label}.tar")
    print(f"{compress_label}: {format_elapsed(time.perf_counter() - start)}")

    shutil.rmtree(out_dir)


def tar_file(src: Path, entry_name: str, dest: Path) -> None:
    with tarfile.open(dest, "w") as archive:
        archive.add(src, arcname=entry_name)


def zip_file(src: Path, entry_name: str, dest: Path) -> None:
    with zipfile.ZipFile(dest, "w", compression=zipfile.ZIP_DEFLATED) as archive:
        archive.write(src, arcname=entry_name)


targets = get_targets()

# Each `go build` parallelizes internally, so cap concurrent targets at the
# CPU count instead of launching all of them at once.
concurrency = min(max(1, os.cpu_count() or 4), len(targets))

start = time.perf_counter()
failures: list[str] = []

with ThreadPoolExecutor(max_workers=concurrency) as executor:
    futures = [executor.submit(build_target, target) for target in targets]
    for future in as_completed(futures):
        try:
            future.result()
        except Exception as err:
            failures.append(str(err))

print(f"Built zvm: {format_elapsed(time.perf_counter() - start)}")

if len(failures) > 0:
    for failure in failures:
        print(failure, file=sys.stderr)
    sys.exit(1)
