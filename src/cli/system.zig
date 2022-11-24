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
    // A switch with an inline else can be used to make special cases for some
    // tags.
    const tag = @tagName(info.target.os.tag);
    return SystemInfo{ .arch = arch, .tag = @as([]const u8, tag) };
}

test "simple test" {
    var list = std.ArrayList(i32).init(std.testing.allocator);
    defer list.deinit(); // try commenting this out and see if zig detects the memory leak!
    try list.append(42);
    try std.testing.expectEqual(@as(i32, 42), list.pop());
}

// This test is only to detect problems for the first implementation
// that made the switching pretty unmaintainable. It will fail as
// soon as the possible targets change, so this test must be deleted then.
//
// Compares the old implementation to the new one, by running both functions for
// the current detected OS.
test "string returned by the conversion to string of the tag" {
    // The old implementation
    const info = try NativeTargetInfo.detect(.{});
    const old_tag = switch (info.target.os.tag) {
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
    const new_func = try getSystemInfo();
    const new_tag = new_func.tag;
    try std.testing.expectEqualStrings(old_tag, new_tag);
}
