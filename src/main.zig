const std = @import("std");
const clap = @import("clap/clap.zig");
const version = @import("fetch-version.zig");
const ArrayList = std.ArrayList;
const NativeTargetInfo = std.zig.system.NativeTargetInfo;

pub fn main() !void {
    const params = comptime clap.parseParamsComptime(
        \\-h, --help             Display this help and exit.
        \\-v, --version          Print out the installed version of zvm.
        \\<str>...
        \\
    );
    // Initalize our diagnostics, which can be used for reporting useful errors.
    // This is optional. You can also pass `.{}` to `clap.parse` if you don't
    // care about the extra information `Diagnostics` provides.
    var diag = clap.Diagnostic{};
    var res = clap.parse(clap.Help, &params, clap.parsers.default, .{
        .diagnostic = &diag,
    }) catch |err| {
        // Report useful error and exit
        diag.report(std.io.getStdErr().writer(), err) catch {};
        return err;
    };
    defer res.deinit();

    if (res.args.help) {
        return clap.help(std.io.getStdErr().writer(), clap.Help, &params, .{});
    } else if (res.args.version) {
        return std.debug.print("zvm (Zig Version Manager) v0.0.1\n", .{});
    }

    // FETCHING DATA
    var arena_state = std.heap.ArenaAllocator.init(std.heap.c_allocator);
    defer arena_state.deinit();

    const allocator = arena_state.allocator();
    var response_buffer = std.ArrayList(u8).init(allocator);

    // superfluous when using an arena allocator, but
    // important if the allocator implementation changes
    defer response_buffer.deinit();

    for (res.positionals) |val, i| {
        if (streql("install", val) and res.positionals.len >= i + 1) {
            try version.fetchVersionJSON(&response_buffer);
            const user_ver = res.positionals[i + 1];
            // std.log.info("Got response of {d} bytes", .{response_buffer.items.len});
            // std.debug.print("{s}\n", .{response_buffer.items});
            const tree = try version.parseVersionJSON(&response_buffer, &arena_state);

            if (tree.root.Object.get(user_ver)) |value| {
                var info = try getSystemInfo();
                var buf: [100]u8 = undefined;
                var buf_slice: []u8 = undefined;

                if (streql(info.arch,  "x86")) {
                    buf_slice = try std.fmt.bufPrint(&buf, "{s}-{s}", .{ "x86_64", info.tag });
                } else {
                    buf_slice = try std.fmt.bufPrint(&buf, "{s}-{s}", .{ info.arch, info.tag });
                }

                std.debug.print("{}", .{value.Object.get(buf_slice).?.Object.get("tarball").?});

            } else {
                std.debug.print("Invalid Zig version provided. Try master or latest-stable\n", .{});
                return;
            }

            return;
        } else if (streql("use", val) and res.positionals.len >= i + 1) {}
    }
}

test "simple test" {
    var list = std.ArrayList(i32).init(std.testing.allocator);
    defer list.deinit(); // try commenting this out and see if zig detects the memory leak!
    try list.append(42);
    try std.testing.expectEqual(@as(i32, 42), list.pop());
}

/// Compares two strings. Returns a bool based on their equality.
fn streql(original: []const u8, compto: []const u8) bool {
    return std.mem.eql(u8, original, compto);
}

const SystemInfo = struct { arch: []const u8, tag: []const u8 };

fn getSystemInfo() !SystemInfo {
    const info = try NativeTargetInfo.detect(.{});
    const arch = info.target.cpu.arch.genericName();
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
