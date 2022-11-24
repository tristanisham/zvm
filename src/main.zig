const std = @import("std");
const ArrayList = std.ArrayList;
const cli = @import("cli/cli.zig");
const string = []const u8;
const ansi = @import("ansi.zig");
const builtin = @import("builtin");

pub fn main() !void {

    // Fetching data. Where we currenlty process the cli.

    // FETCHING DATA
    var arena_state = std.heap.ArenaAllocator.init(std.heap.c_allocator);
    defer arena_state.deinit();

    var allocator = arena_state.allocator();

    var args = cli.Args{};
    try args.parse(allocator);

    if (args.positionals == null) {
        return std.debug.print("{s}", .{args.help});
    }

    if (args.positionals) |argv| {
        // Command line parser
        for (argv) |val, i| {
            if (std.mem.eql(u8, "install", val) and argv.len >= i + 1) {
                try cli.install.install(&allocator, argv[i + 1]);
                // try version.fetchVersionJSON(&response_buffer);
                // const user_ver = argv[i + 1];
                // // std.log.info("Got response of {d} bytes", .{response_buffer.items.len});
                // // std.debug.print("{s}\n", .{response_buffer.items});
                // const tree = try version.parseVersionJSON(&response_buffer, &arena_state);

                // if (tree.root.Object.get(user_ver)) |value| {
                //     var info = try getSystemInfo();
                //     var zig_ver_slice: []u8 = undefined;

                //     if (std.mem.eql(u8, info.arch, "x86")) {
                //         zig_ver_slice = try std.fmt.allocPrint(allocator, "{s}-{s}", .{ "x86_64", info.tag });
                //     } else {
                //         zig_ver_slice = try std.fmt.allocPrint(allocator, "{s}-{s}", .{ info.arch, info.tag });
                //     }

                //     const tarball: []const u8 = value.Object.get(zig_ver_slice).?.Object.get("tarball").?.String;
                //     const data = try std.mem.Allocator.dupeZ(allocator, u8, tarball);

                //     const USER_HOME = cli.install.homeDir(allocator) orelse "~";
                //     const zvm_dir = try std.fmt.allocPrint(allocator, "{s}/.zvm", .{USER_HOME});
                //     const home = try std.fs.openDirAbsolute(cli.install.homeDir(allocator) orelse "~", .{});

                //     home.makeDir(".zvm") catch |err| {
                //         switch (err) {
                //             error.PathAlreadyExists => std.debug.print("Installing {s} in {s}\n", .{ user_ver, zvm_dir }),
                //             else => return err,
                //         }
                //     };

                //     // ~/.zvm/
                //     const pre_out_path = try std.fmt.allocPrintZ(allocator, "{s}/zig-{s}-{s}", .{ zvm_dir, user_ver, zig_ver_slice });

                //     const extension = switch (builtin.os.tag) {
                //         .windows => "zip",
                //         else => "tar.xz",
                //     };

                //     const out_path: [:0]u8 = try std.fmt.allocPrintZ(allocator, "{s}.{s}", .{ pre_out_path, extension });

                //     const untar_path: [:0]u8 = try std.fmt.allocPrintZ(allocator, "{s}/zig-{s}-{s}", .{ zvm_dir, user_ver, zig_ver_slice });
                //     try version.downloadFile(data, out_path);

                //     if (args.outpath != null) {
                //         std.debug.print("\x1b[{s}m-o flag is an alpha feature and currently does nothing. It might be depreciated at any time.\x1b[0m\n", .{ansi.darkYellow});
                //     }

                //     var env_map = try std.process.getEnvMap(allocator);

                //     // Creates the path for tar to extract into
                //     std.fs.makeDirAbsolute(untar_path) catch |err| {
                //         switch (err) {
                //             error.PathAlreadyExists => std.debug.print("Untarring {s} in {s}\n", .{ user_ver, zvm_dir }),
                //             else => return err,
                //         }
                //     };

                //     var uncompress_proc: std.ChildProcess.ExecResult = switch (builtin.os.tag) {
                //         .windows => try std.ChildProcess.exec(.{
                //             .argv = &.{ "Expand-Archive", out_path, "-DestinationPath", untar_path },
                //             .allocator = allocator,
                //             .env_map = &env_map,
                //         }),
                //         else => try std.ChildProcess.exec(.{
                //             .argv = &.{ "tar", "-xf", out_path, "-C", untar_path },
                //             .allocator = allocator,
                //             .env_map = &env_map,
                //         }),
                //     };
                //     if (uncompress_proc.stderr.len > 0) std.debug.print("\x1b[{s}mThere was an error calling `tar` on your system path:\n\n{s}\x1b[{s}m\n", .{ ansi.darkRed, uncompress_proc.stderr, ansi.reset });
            } else if (std.mem.eql(u8, "use", val) and argv.len >= i + 1) {} else if (std.mem.eql(u8, "upgrade", val) and argv.len >= i + 1) {
                std.debug.print("upgrade called\n", .{});
                std.debug.print("{s}", .{cli.system.homeDir(allocator).?});
            } else {
                std.debug.print("Invalid Zig version provided. Try master\n", .{});
                return;
            }

            return;
        }
    }
}



test "simple test" {
    var list = std.ArrayList(i32).init(std.testing.allocator);
    defer list.deinit(); // try commenting this out and see if zig detects the memory leak!
    try list.append(42);
    try std.testing.expectEqual(@as(i32, 42), list.pop());
}
