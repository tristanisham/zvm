const std = @import("std");
const builtin = @import("builtin");

pub const Zvm = struct {
    zvm_dir: []const u8 = "~/.zvm",
    alloc: std.mem.Allocator,
    settings: Settings = Settings{
        .useColor = true,
    },

    pub fn init(alloc: std.mem.Allocator) !Zvm {
        var zvm = Zvm{
            .alloc = alloc,
        };
        var home: []const u8 = undefined;
        if (builtin.os.tag == .windows) {
            home = std.os.getenv("USERPROFILE") orelse "~/";
        } else {
            home = std.os.getenv("HOME") orelse "~/";
        }

        zvm.zvm_dir = try std.fs.path.join(alloc, &.{ home, ".zvm" });
        try std.fs.cwd().makePath(zvm.zvm_dir);
        try zvm.loadSettings();

        return zvm;
    }

    pub fn loadSettings(self: *Zvm) !void {
        const settings_path = try std.fs.path.join(self.alloc, &.{ self.zvm_dir, "settings.json" });
        const settings = std.fs.cwd().openFile(settings_path, .{}) catch |err| {
            if (err == std.fs.File.OpenError.FileNotFound) {
                var string = std.ArrayList(u8).init(self.alloc);
                try std.json.stringify(self.settings, .{ .whitespace = .{} }, string.writer()); // Figure out padding setting
                try std.fs.cwd().writeFile(settings_path, string.items);
                return;
            }

            return err;
        };
        defer settings.close();

        const setting_data = try settings.readToEndAlloc(self.alloc, 4096);
        var token_stream = std.json.TokenStream.init(setting_data);
        self.settings = try std.json.parse(Settings, &token_stream, .{
            .ignore_unknown_fields = true,
        });
    }

    /// Has not been implemented. Will do nothing.
    pub fn install(self: *Zvm,version: []const u8) !void {
        _ = self;
        _ = version;
    }

    /// Has not been implemented. Will do nothing.
    pub fn use(self: *Zvm,version: []const u8) !void {
        _ = self;
        _ = version;
    }

    /// Has not been implemented. Will do nothing.
    pub fn listVersions(self: *Zvm,) !void {
        _ = self;

    }

    /// Has not been implemented. Will do nothing.
    pub fn uninstall(self: *Zvm,version: []const u8) !void {
        _ = self;
        _ = version;

    }
};

pub const Settings = struct {
    useColor: bool,
};

pub const Args = enum {
    install,
    i,
    use,
    ls,
    uninstall,
    rm,
    help,

    pub fn printHelp() void {
        std.debug.print("Hi\n", .{});
    }
};