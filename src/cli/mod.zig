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
    pub fn install(self: *Zvm, version: []const u8) !void {
        var client = std.http.Client{ .allocator = self.alloc };

        const uri = try std.Uri.parse("https://ziglang.org/download/index.json");
        var headers = std.http.Headers{ .allocator = self.alloc };
        defer headers.deinit();

        try headers.append("accept", "application/json"); // tell the server we'll accept anything
        var req = client.request(.GET, uri, headers, .{}) catch std.debug.panic("Failed to create http request", .{});
        defer req.deinit();

        try req.start();
        try req.wait();
        const content_len = req.response.headers.getFirstValue("content-length") orelse "34010";

        const body = try req.reader().readAllAlloc(self.alloc, std.fmt.parseInt(usize, content_len, 10) catch unreachable);
        defer self.alloc.free(body);

        // std.debug.print("{s}", .{body});

        var parser = std.json.Parser.init(self.alloc, false);
        defer parser.deinit();

        var tree = try parser.parse(body);

        var user_req_distro = tree.root.Object.get(version) orelse {
            std.debug.print("Invalid version: {s}\n", .{version});
            for (tree.root.Object.iterator().keys.*) |key| {
                std.debug.print("{s}\n", .{key});
            }
            std.process.exit(1);
        };

        const target = try std.fmt.allocPrint(self.alloc, "{s}-{s}", .{ @tagName(builtin.cpu.arch), @tagName(builtin.os.tag) });

        std.debug.print("{s}", .{target});
        // Select's versions of Zig according to runtime platform.
        // Example:
        //  "x86_64-macos": {
        //    "tarball": "https://ziglang.org/builds/zig-macos-x86_64-0.11.0-dev.3045+526065723.tar.xz",
        //    "shasum": "fa29a1cf25376db30059b800812e1484141b47a94710e8e56fe443cd20eba498",
        //    "size": "46621376"
        //  },
        var user_req_platform = user_req_distro.Object.get(target) orelse {
            std.debug.print("Your system is currenlty not supported: {s}\n", .{target});
            std.process.exit(1);
        };

        user_req_platform.dump();
    }

    /// Has not been implemented. Will do nothing.
    pub fn use(self: *Zvm, version: []const u8) !void {
        _ = self;
        _ = version;
    }

    /// Has not been implemented. Will do nothing.
    pub fn listVersions(
        self: *Zvm,
    ) !void {
        _ = self;
    }

    /// Has not been implemented. Will do nothing.
    pub fn uninstall(self: *Zvm, version: []const u8) !void {
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
