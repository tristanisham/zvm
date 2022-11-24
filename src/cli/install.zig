const std = @import("std");
const builtin = @import("builtin");
const curl = @import("../curl.zig");
const sys = @import("system.zig");
const ansi = @import("../ansi.zig");



pub fn install(allocator: *std.mem.Allocator, version: [:0]u8) !void {
    // var allocator = arena_state.allocator();
    
    var response_buffer = std.ArrayList(u8).init(allocator.*);

    // superfluous when using an arena allocator, but
    // important if the allocator implementation changes
    defer response_buffer.deinit();
    try curl.fetchVersionJSON(&response_buffer);
    const user_ver = version;
    // std.log.info("Got response of {d} bytes", .{response_buffer.items.len});
    // std.debug.print("{s}\n", .{response_buffer.items});
    const tree = try curl.parseVersionJSON(&response_buffer, allocator);

    if (tree.root.Object.get(user_ver)) |value| {
        var info = try sys.getSystemInfo();
        var zig_ver_slice: []u8 = undefined;

        if (std.mem.eql(u8, info.arch, "x86")) {
            zig_ver_slice = try std.fmt.allocPrint(allocator.*, "{s}-{s}", .{ "x86_64", info.tag });
        } else {
            zig_ver_slice = try std.fmt.allocPrint(allocator.*, "{s}-{s}", .{ info.arch, info.tag });
        }

        const tarball: []const u8 = value.Object.get(zig_ver_slice).?.Object.get("tarball").?.String;
        const data = try std.mem.Allocator.dupeZ(allocator.*, u8, tarball);

        const USER_HOME = sys.homeDir(allocator.*) orelse "~";
        const zvm_dir = try std.fmt.allocPrint(allocator.*, "{s}/.zvm", .{USER_HOME});
        const home = try std.fs.openDirAbsolute(sys.homeDir(allocator.*) orelse "~", .{});

        home.makeDir(".zvm") catch |err| {
            switch (err) {
                error.PathAlreadyExists => std.debug.print("Installing {s} in {s}\n", .{ user_ver, zvm_dir }),
                else => return err,
            }
        };

        // ~/.zvm/
        const pre_out_path = try std.fmt.allocPrintZ(allocator.*, "{s}/zig-{s}-{s}", .{ zvm_dir, user_ver, zig_ver_slice });

        const extension = switch (builtin.os.tag) {
            .windows => "zip",
            else => "tar.xz",
        };

        const out_path: [:0]u8 = try std.fmt.allocPrintZ(allocator.*, "{s}.{s}", .{ pre_out_path, extension });

        const untar_path: [:0]u8 = try std.fmt.allocPrintZ(allocator.*, "{s}/zig-{s}-{s}", .{ zvm_dir, user_ver, zig_ver_slice });
        try curl.downloadFile(data, out_path);

        // if (args.outpath != null) {
        //     std.debug.print("\x1b[{s}m-o flag is an alpha feature and currently does nothing. It might be depreciated at any time.\x1b[0m\n", .{ansi.darkYellow});
        // }

        var env_map = try std.process.getEnvMap(allocator.*);

        // Creates the path for tar to extract into
        std.fs.makeDirAbsolute(untar_path) catch |err| {
            switch (err) {
                error.PathAlreadyExists => std.debug.print("Untarring {s} in {s}\n", .{ user_ver, zvm_dir }),
                else => return err,
            }
        };

        var uncompress_proc: std.ChildProcess.ExecResult = switch (builtin.os.tag) {
            .windows => try std.ChildProcess.exec(.{
                .argv = &.{ "Expand-Archive", out_path, "-DestinationPath", untar_path },
                .allocator = allocator.*,
                .env_map = &env_map,
            }),
            else => try std.ChildProcess.exec(.{
                .argv = &.{ "tar", "-xf", out_path, "-C", untar_path },
                .allocator = allocator.*,
                .env_map = &env_map,
            }),
        };
        if (uncompress_proc.stderr.len > 0) std.debug.print("\x1b[{s}mThere was an error calling `tar` on your system path:\n\n{s}\x1b[{s}m\n", .{ ansi.darkRed, uncompress_proc.stderr, ansi.reset });
    } else {
        std.debug.print("Invalid Zig version provided. Try master\n", .{});
        return;
    }

    return;
}
