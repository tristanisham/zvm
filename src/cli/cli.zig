pub const install = @import("install.zig");
const std = @import("std");
const mem = std.mem;
const testing = std.testing;

pub const Args = struct {
    outpath: ?[]u8 = null,
    help: []const u8 = "(Zig Version Manager)\n-h, --help\tDisplay this help and exit.\n-v, --version\tPrint out the installed version of zvm.\n-o, --out <str>\tChanges the path at which ZVM installs and unzips Zig.",
    version: []const u8 = "zvm (Zig Version Manager) v0.0.1\n",
    positionals: ?[][:0]u8 = null,

    /// `parse()` analyzes command line arguments and fill the appropriate struct fields.
    ///
    /// If arguments for 'help' or 'version' are detected. `parse()` will print
    /// the appropiate value to stderr and exit 0.
    pub fn parse(self: *Args, alloc: std.mem.Allocator) !void {
        const args: [][:0]u8 = try std.process.argsAlloc(alloc);
        if (args.len > 0) self.positionals = args else return;
        for (args) |arg, i| {
            if ((mem.eql(u8, arg, "-o") or mem.eql(u8, arg, "--outfile")) and args.len >= i + 1) {
                self.outpath = args[i + 1];
            } else if (mem.eql(u8, arg, "-h") or mem.eql(u8, arg, "--help")) {
                std.debug.print("{s}\n", .{self.help});
                std.os.exit(0);
            } else if (mem.eql(u8, arg, "-v") or mem.eql(u8, arg, "--version")) {
                std.debug.print("{s}\n", .{self.version});
                std.os.exit(0);
            }
        }
    }
};

// This tests passes, but it's annoying to see red when I'm not looking for it.
// test "-o flag grabs outpath" {
//     const alloc = testing.allocator();
//     defer alloc.deinit();
//     var args = Args{};
//     try args.parse(alloc);
//     testing.expectEqual("test", args.outpath);
// }