const std = @import("std");
const sys = @import("system.zig");
const curl = @import("../curl.zig");



pub fn use(alloc: *std.mem.Allocator, version: []const u8) !void {
    const USER_HOME = sys.homeDir(alloc.*) orelse "~";
    const zvm = std.fs.openDirAbsolute(try std.fmt.allocPrint(alloc.*, "{s}/.zvm", .{USER_HOME}), .{}) catch |err| {
        switch (err) {
            error.FileNotFound => std.debug.panic("Required directory ~/.zvm not found. Please run ", .{}),
            else => return err
        }
    };

    const version_file = try zvm.openFile("versions.json", .{});
    var data: []u8 = try version_file.readToEndAlloc(alloc.*, 4096);
    const json = try curl.parseVersionJSON(&data, alloc);
    if (json.root.Object.get(version) == null) {
        return error.InvalidZigVersion;
    }

    zvm.deleteFile("bin") catch |err| {
        switch (err) {
            error.FileNotFound, error.Unexpected => {},
            else => return err
        }
    };

    try std.fs.symLinkAbsolute(try std.fmt.allocPrint(alloc.*, "{s}/.zvm/bin", .{USER_HOME}), try std.fmt.allocPrint(alloc.*, "{s}/.zvm/{s}", .{USER_HOME, version}), .{});
    
}
