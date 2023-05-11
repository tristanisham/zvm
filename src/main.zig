const std = @import("std");
const cli = @import("cli/mod.zig");

pub fn main() !void {
    var arena = std.heap.ArenaAllocator.init(std.heap.page_allocator);
    defer arena.deinit();

    const allocator = arena.allocator();

    var zvm = try cli.Zvm.init(allocator);

    const args = try std.process.argsAlloc(allocator);
    defer std.process.argsFree(allocator, args);

    if (args.len == 1) {
        cli.Args.printHelp();
    }

    //TODO: Consider switching to while loop for greater control.
    for (args[1..], 1..) |arg, i| {
        switch (std.meta.stringToEnum(cli.Args, arg) orelse continue) {
            .install, .i => {
                if (args.len > i + 1) {
                    const version = extractVersion(args[i + 1]);
                    try zvm.install(version);
                }
            },
            .use => {
                if (args.len > i + 1) {
                    const version = extractVersion(args[i + 1]);
                    try zvm.use(version);
                }
            },
            .uninstall, .rm => {
                if (args.len > i + 1) {
                    const version = extractVersion(args[i + 1]);
                    try zvm.uninstall(version);
                }
            },
            .ls => {
                try zvm.listVersions();
            },
            .help => cli.Args.printHelp(),
        }
    }
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
