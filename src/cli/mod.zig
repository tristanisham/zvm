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

        var user_req_distro = tree.root.Object.get(version) orelse
            return self.tryFetchUserVersion(version) catch {
            std.debug.print("Couldn't fetch version: {s}\n", .{version});
            std.debug.print("Available versions:\n", .{});
            var iter = tree.root.Object.iterator();
            while (iter.next()) |entry| {
                std.debug.print("{s}\n", .{entry.key_ptr.*});
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

    fn tryFetchUserVersion(self: Zvm, version: []const u8) !void {
        var client = std.http.Client{ .allocator = self.alloc };
        defer client.deinit();
        var headers = std.http.Headers{ .allocator = self.alloc };
        defer headers.deinit();
        const url = try getDefaultUrl(self.alloc, version);
        std.debug.print("Fetching {s}\n", .{url});
        const uri = try std.Uri.parse(url);

        var req = try client.request(.GET, uri, headers, .{});
        defer req.deinit();

        try req.start();
        try req.wait();

        // found user provided version
        std.debug.print("Installing {s} to {s}\n", .{ version, self.zvm_dir });
        const version_path = try std.fs.path.join(self.alloc, &.{ self.zvm_dir, version });
        const dir = try std.fs.cwd().makeOpenPath(version_path, .{});
        const content_type = req.response.headers.getFirstValue("Content-Type") orelse unreachable;
        if (std.ascii.eqlIgnoreCase(content_type, "application/gzip") or
            std.ascii.eqlIgnoreCase(content_type, "application/x-gzip"))
            try unpackTarball(self.alloc, &req, dir, std.compress.gzip)
        else if (std.ascii.eqlIgnoreCase(content_type, "application/x-xz"))
            try unpackTarball(self.alloc, &req, dir, std.compress.xz)
        else {
            std.log.err("unexpected content-type '{s}'", .{content_type});
            std.log.err("body: ", .{});
            var buf: [std.mem.page_size]u8 = undefined;
            while (true) {
                const amt = try req.read(&buf);
                if (amt == 0) break;
                std.debug.print("{s}", .{buf[0..amt]});
            }
            return error.ContentType; // TODO handle other compression type
        }
    }

    // from https://github.com/ziglang/zig/blob/master/src/Package.zig#L560
    fn unpackTarball(
        alloc: std.mem.Allocator,
        req: *std.http.Client.Request,
        dir: std.fs.Dir,
        comptime compression: type,
    ) !void {
        var br = std.io.bufferedReaderSize(std.crypto.tls.max_ciphertext_record_len, req.reader());
        var decompress = try compression.decompress(alloc, br.reader());
        defer decompress.deinit();
        // BOOM seeing same error trace as in https://github.com/ziglang/zig/issues/15590
        try std.tar.pipeToFileSystem(dir, decompress.reader(), .{ .mode_mode = .ignore });
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

// begin traviss yoink from https://github.com/marler8997/zigup/blob/a16b61e74e47b3dbd7be45ba1dcdf8a1d259a8c5/zigup.zig#L15
const arch = switch (builtin.cpu.arch) {
    .x86_64 => "x86_64",
    .aarch64 => "aarch64",
    .riscv64 => "riscv64",
    else => @compileError("Unsupported CPU Architecture"),
};
const os = switch (builtin.os.tag) {
    .windows => "windows",
    .linux => "linux",
    .macos => "macos",
    else => @compileError("Unsupported OS"),
};
const url_platform = os ++ "-" ++ arch;
const json_platform = arch ++ "-" ++ os;
const archive_ext = if (builtin.os.tag == .windows) "zip" else "tar.xz";

const VersionKind = enum { release, dev };
fn determineVersionKind(version: []const u8) VersionKind {
    return if (std.mem.indexOfAny(u8, version, "-+")) |_| .dev else .release;
}

fn getDefaultUrl(allocator: std.mem.Allocator, compiler_version: []const u8) ![]const u8 {
    return switch (determineVersionKind(compiler_version)) {
        .dev => try std.fmt.allocPrint(allocator, "https://ziglang.org/builds/zig-" ++ url_platform ++ "-{0s}." ++ archive_ext, .{compiler_version}),
        .release => try std.fmt.allocPrint(allocator, "https://ziglang.org/download/{s}/zig-" ++ url_platform ++ "-{0s}." ++ archive_ext, .{compiler_version}),
    };
}
// end traviss yoink

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
    unknown,

    pub fn printHelp(comptime fmt: []const u8, args: anytype) void {
        std.debug.print(fmt, args);
        std.debug.print("usage: ...", .{});
    }
};
