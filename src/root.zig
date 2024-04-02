const std = @import("std");
const testing = std.testing;
const Allocator = std.mem.Allocator;
const builtin = @import("builtin");

pub const ZVM = struct {
    alloc: Allocator,
    home_dir: []const u8,
    base_dir: ?std.fs.Dir = null,
    settings: ?Settings = null,

    pub fn install(self: *@This(), version: []const u8) !void {
        std.debug.print("version: {s}\n", .{version});
        var client = std.http.Client{ .allocator = self.alloc };
        defer client.deinit();

        // if (self.settings.?.versionMapUrl) |url| {
        //     _ = std.Uri.parse(url) catch |err| {
        //         std.debug.print("{s}: {any}", .{ @errorName(err), self.settings.?.versionMapUrl });
        //         std.process.exit(1);
        //     };
        // }

        try self.loadSettings();
        std.debug.print("vmu: {?s}\n", .{self.settings.?.versionMapUrl});
        if (self.settings) |settings| {
            const vmu = settings.versionMapUrl orelse "https://ziglang.org/download/index.json";
            var buff = std.ArrayList(u8).init(self.alloc);
            defer buff.deinit();

            // std.debug.print("version map: {s}", .{vm});
            const resp = try client.fetch(.{
                .location = .{
                    .url = vmu,
                },
                .headers = .{
                    .accept_encoding = "application/json"
                },
                .response_storage = .{ .dynamic = &buff },
            });

            if (resp.status.class() != .success) {
                std.debug.print("Error fetching vmu: {?s}\n", .{resp.status.phrase()});
                std.process.exit(1);
            }

            std.debug.print("len {d}, {s}", .{ buff.items.len, buff.items });
        }

        // const vMap = try std.json.parseFromSlice(std.json.ObjectMap, self.alloc, buff.items, .{ .ignore_unknown_fields = true });
        // defer vMap.deinit();
        // // const os_str = @tagName(builtin.os.tag);
        // // const arch_str = builtin.cpu.arch.genericName();
        // if (streql(version, "master")) {
        //     // if (vmu.value.get(version)) |value| {
        //     //             value.
        //     //         }

        // } else {
        //     if (vMap.value.get(version)) |value| {
        //         std.debug.print("{any}", .{value.dump()});
        //     //     var arch_os_buff: [50]u8 = undefined;

        //     //     const name = try std.fmt.bufPrint(&arch_os_buff, "{s}-{s}", .{arch_str, os_str});
        //     //     if (value.object.get(name)) |dist| {
        //     //         if (dist.object.get("tarball")) |tarball| {
        //     //             std.debug.print("{s}", .{tarball.string});
        //     //         }
        //     //     }
        //     }
        // }
    }

    pub fn initWithAllocator(allocator: Allocator) !@This() {
        var zvm = ZVM{
            .alloc = allocator,
            .home_dir = "~/",
        };

        var env_vars = try std.process.getEnvMap(zvm.alloc);
        defer env_vars.deinit();
        if (builtin.os.tag != .windows) {
            zvm.home_dir = env_vars.get("HOME") orelse "~/";
        } else {
            zvm.home_dir = "%userprofile%";
        }

        if (env_vars.get("ZVM_PATH")) |zvm_path| {
            _ = std.fs.cwd().makeDir(zvm_path) catch |err| {
                switch (err) {
                    error.PathAlreadyExists => {},
                    else => return err,
                }
            };

            zvm.base_dir = try std.fs.openDirAbsolute(zvm_path, .{});
        } else {
            const paths = try std.fs.path.join(zvm.alloc, &.{ zvm.home_dir, ".zvm" });
            defer zvm.alloc.free(paths);
            _ = std.fs.cwd().makeDir(paths) catch |err| {
                switch (err) {
                    error.PathAlreadyExists => {},
                    else => return err,
                }
            };

            zvm.base_dir = try std.fs.openDirAbsolute(paths, .{});
        }

        if (zvm.settings == null) {
            try zvm.loadSettings();
        }

        return zvm;
    }

    pub fn createSettings(self: *@This()) !std.fs.File {
        const settings = try self.base_dir.?.createFile("settings.json", .{ .read = true });
        try self.saveSettings();
        return settings;
    }

    pub fn loadSettings(
        self: *@This(),
    ) !void {
        var file: std.fs.File = undefined;
        if (self.base_dir.?.openFile("./settings.json", .{})) |data| {
            file = data;
        } else |err| {
            switch (err) {
                // error.PathAlreadyExists => {},
                error.FileNotFound => file = try self.createSettings(),
                else => return err,
            }
        }

        const stats = try file.stat();

        if (stats.size == 0) {
            std.debug.print("file is empty", .{});
        }

        const data = try file.readToEndAlloc(self.alloc, stats.size);
        defer self.alloc.free(data);

        const parsed = try std.json.parseFromSlice(Settings, self.alloc, data, .{});
        defer parsed.deinit();

        var v_url_buff: [std.fs.MAX_PATH_BYTES]u8 = undefined;
        var fba = std.heap.FixedBufferAllocator.init(&v_url_buff);

        if (parsed.value.versionMapUrl) |vurl| {
            const dupe = try fba.allocator().dupe(u8, vurl);
            self.settings = .{
                .useColor = parsed.value.useColor,
                .versionMapUrl = dupe,
            };
        } else {
            self.settings = .{
                .useColor = parsed.value.useColor,
                .versionMapUrl = null,
            };
        }
    }

    pub fn deinit(self: *@This()) void {
        self.base_dir.?.close();
    }

    pub fn saveSettings(self: *@This()) !void {
        var buf: [100]u8 = undefined;
        var fba = std.heap.FixedBufferAllocator.init(&buf);
        var string = std.ArrayList(u8).init(fba.allocator());
        try std.json.stringify(self.settings, .{}, string.writer());
        try self.base_dir.?.writeFile("./settings.json", string.items);
    }
};

pub fn streql(a: []const u8, b: []const u8) bool {
    return std.mem.eql(u8, a, b);
}

pub const Settings = struct {
    useColor: bool = true,
    versionMapUrl: ?[]u8,
};

pub const SettingsError = error{
    MissingSettingsFile,
};
