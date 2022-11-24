const std = @import("std");
const NativeTargetInfo = std.zig.system.NativeTargetInfo;

const SystemInfo = struct { arch: []const u8, tag: []const u8 };

/// Fetches the user's home directory.
// Pulls in $HOME. More needed to determine if valid on Windows.
pub fn homeDir(alloc: std.mem.Allocator) ?[]u8 {
    return std.process.getEnvVarOwned(alloc, "HOME") catch null;
}

pub fn getSystemInfo() !SystemInfo {
    const info = try NativeTargetInfo.detect(.{});
    const arch = info.target.cpu.arch.genericName();
    // https://discord.com/channels/605571803288698900/1019652020308824145
    const tag = switch (info.target.os.tag) {
        .ananas => "ananas",
        .cloudabi => "cloudabi",
        .dragonfly => "dragonfly",
        .freebsd => "freebsd",
        .fuchsia => "fuchsia",
        .ios => "ios",
        .kfreebsd => "kfreebsd",
        .linux => "linux",
        .lv2 => "lv2",
        .macos => "macos",
        .netbsd => "netbsd",
        .openbsd => "openbsd",
        .solaris => "solaris",
        .windows => "windows",
        .zos => "zos",
        .haiku => "haiku",
        .minix => "minix",
        .rtems => "rtems",
        .nacl => "nacl",
        .aix => "aix",
        .cuda => "cuda",
        .nvcl => "nvcl",
        .amdhsa => "amdhsa",
        .ps4 => "ps4",
        .ps5 => "ps5",
        .elfiamcu => "elfiamcu",
        .tvos => "tvos",
        .watchos => "watchos",
        .driverkit => "driverkit",
        .mesa3d => "mesa3d",
        .contiki => "contiki",
        .amdpal => "amdpal",
        .hermit => "hermit",
        .hurd => "hurd",
        .wasi => "wasi",
        .emscripten => "emscripten",
        .shadermodel => "shadermodel",
        .uefi => "uefi",
        .opencl => "opencl",
        .glsl450 => "glsl450",
        .vulkan => "vulkan",
        .plan9 => "plan9",
        .other => "other",
        .freestanding => "freestanding",
    };

    return SystemInfo{ .arch = arch, .tag = @as([]const u8, tag) };
}
