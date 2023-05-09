const std = @import("std");
const builtin = @import("builtin");

pub const Zvm = struct {
    zvm_dir: []const u8 = "~/.zvm",
    alloc: std.mem.Allocator,
    settings: Settings = Settings{
        .use_color = true,
        .base_path = "~/.zvm/settings.json",
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

        try zvm.loadSettings();

        return zvm;
    }

    pub fn loadSettings(self: *Zvm) !void {
        const settings_path = try std.fs.path.join(self.alloc, &.{ self.zvm_dir, "settings.json" });
        self.settings.base_path = settings_path;
        const settings = std.fs.cwd().openFile(settings_path, .{}) catch |err| {
            if (err == std.fs.File.OpenError.FileNotFound) {

                var string = std.ArrayList(u8).init(self.alloc);
                try std.json.stringify(self.settings, .{}, string.writer()); // Figure out padding setting
                try std.fs.cwd().writeFile(settings_path, string.items);
                return;
            }

            return err;
        };
        defer settings.close();

        const setting_data = try settings.readToEndAlloc(self.alloc, 4096);
        var token_stream = std.json.TokenStream.init(setting_data);
        self.settings = try std.json.parse(Settings, &token_stream, .{});
    }
};

pub const Settings = struct {
    base_path: []const u8,
    use_color: bool,
};
