const std = @import("std");
const cli = @import("cli/mod.zig");

pub fn main() !void {
    var arena = std.heap.ArenaAllocator.init(std.heap.page_allocator);
    defer arena.deinit();

    const allocator = arena.allocator();

    var zvm = try cli.Zvm.init(allocator);

    var args = try std.process.argsAlloc(allocator);
    defer std.process.argsFree(allocator, args);

    if (args.len == 1) {
        return cli.Args.printHelp("", .{});
    }

    var i: usize = 1;
    while (i < args.len) : (i += 1) {
        const arg = std.meta.stringToEnum(cli.Args, args[i]) orelse .unknown;
        switch (arg) {
            .install, .i => {
                const version = nextArg(&i, args) orelse
                    return cli.Args.printHelp("{s} missing version", .{@tagName(arg)});
                try zvm.install(version);
                break;
            },
            .use => {
                const version = nextArg(&i, args) orelse
                    return cli.Args.printHelp("{s} missing version", .{@tagName(arg)});
                try zvm.use(version);
                break;
            },
            .uninstall, .rm => {
                const version = nextArg(&i, args) orelse
                    return cli.Args.printHelp("{s} missing version", .{@tagName(arg)});
                try zvm.uninstall(version);
                break;
            },
            .ls => {
                try zvm.listVersions();
                break;
            },
            .unknown => {
                cli.Args.printHelp("unknown arg {s}\n", .{args[i]});
                break;
            },
            .help => {
                cli.Args.printHelp("", .{});
                break;
            },
        }
    }
}

fn nextArg(i: *usize, args: []const []const u8) ?[]const u8 {
    if (i.* >= args.len) return null;
    i.* += 1;
    return args[i.*];
}

fn extractVersion(v: []const u8) []const u8 {
    return std.mem.trimLeft(u8, v, "v");
}

test "load settings" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();
    const allocator = arena.allocator();
    const zvm = try cli.Zvm.init(allocator);
    try std.testing.expectEqual(zvm.settings, cli.Settings{ .useColor = true });
}
