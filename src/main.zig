const std = @import("std");
const clap = @import("clap/clap.zig");
const version = @import("fetch-version.zig");
const ArrayList = std.ArrayList;
const NativeTargetInfo = std.zig.system.NativeTargetInfo;
const cli = @import("cli/cli.zig");

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

    // Fetching data. Where we currenlty process the cli.

    // FETCHING DATA
    var arena_state = std.heap.ArenaAllocator.init(std.heap.c_allocator);
    defer arena_state.deinit();

    const allocator = arena_state.allocator();
    var response_buffer = std.ArrayList(u8).init(allocator);

    // superfluous when using an arena allocator, but
    // important if the allocator implementation changes
    defer response_buffer.deinit();

    if (res.positionals.len == 0) {
        return clap.help(std.io.getStdErr().writer(), clap.Help, &params, .{});
    }

    // Command line parser
    for (res.positionals) |val, i| {
        if (streql("install", val) and res.positionals.len >= i + 1) {
            try version.fetchVersionJSON(&response_buffer);
            const user_ver = res.positionals[i + 1];
            // std.log.info("Got response of {d} bytes", .{response_buffer.items.len});
            // std.debug.print("{s}\n", .{response_buffer.items});
            const tree = try version.parseVersionJSON(&response_buffer, &arena_state);

            if (tree.root.Object.get(user_ver)) |value| {
                var info = try getSystemInfo();
                var buf_slice: []u8 = undefined;

                if (streql(info.arch, "x86")) {
                    buf_slice = try std.fmt.allocPrint(allocator, "{s}-{s}", .{ "x86_64", info.tag });
                } else {
                    buf_slice = try std.fmt.allocPrint(allocator, "{s}-{s}", .{ info.arch, info.tag });
                }

                const tarball: []const u8 = value.Object.get(buf_slice).?.Object.get("tarball").?.String;
                const data = try std.mem.Allocator.dupeZ(allocator, u8, tarball);

                const USER_HOME = cli.install.homeDir(allocator) orelse "~";
                const zvm_dir = try std.fmt.allocPrint(allocator, "{s}/.zvm", .{USER_HOME});
                const home = try std.fs.openDirAbsolute(cli.install.homeDir(allocator) orelse "~", .{});

                home.makeDir(".zvm") catch |err| {
                    switch (err) {
                        error.PathAlreadyExists => std.debug.print("Installing {s} in {s}\n", .{ user_ver, zvm_dir }),
                        else => return err,
                    }
                };

                const out_path: [:0]u8 = try std.fmt.allocPrintZ(allocator, "{s}/zig-{s}-{s}.tar.xz", .{ zvm_dir, user_ver, buf_slice });
                try version.downloadFile(data, out_path);

                const args = [_:null]?[*:0]const u8{ try std.fmt.allocPrintZ(allocator, "-xf={s}", .{out_path.ptr})};
                const envp = [_:null]?[*:0]const u8{null};

                // for (args) |x| {
                //     std.debug.print("{s}\n", .{x.?});
                // }

                const exec_err = std.os.execvpeZ("tar", args[0..], envp[0..]);
                switch (exec_err) {
                    error.Unexpected => std.debug.print("Succsessfully extracted Zig download", .{}),
                    else => std.debug.panic("{any}", .{exec_err}),
                }
            } else {
                std.debug.print("Invalid Zig version provided. Try master\n", .{});
                return;
            }

            return;
        } else if (streql("use", val) and res.positionals.len >= i + 1) {} else if (streql("upgrade", val) and res.positionals.len >= i + 1) {
            std.debug.print("upgrade called\n", .{});
            std.debug.print("{s}", .{cli.install.homeDir(allocator).?});
        }
    }
}

/// Compares two strings. Returns a bool based on their equality.
fn streql(original: []const u8, compto: []const u8) bool {
    return std.mem.eql(u8, original, compto);
}

const SystemInfo = struct { arch: []const u8, tag: []const u8 };

fn getSystemInfo() !SystemInfo {
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

test "simple test" {
    var list = std.ArrayList(i32).init(std.testing.allocator);
    defer list.deinit(); // try commenting this out and see if zig detects the memory leak!
    try list.append(42);
    try std.testing.expectEqual(@as(i32, 42), list.pop());
}
