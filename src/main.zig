const std = @import("std");
const lib = @import("root.zig");

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    const alloc = gpa.allocator();
    defer {
        const deinit_status = gpa.deinit();
        if (deinit_status == .leak) @panic("MEMORY LEAK (main() gpa)");
    }

    var iter = try std.process.argsWithAllocator(alloc);
    defer iter.deinit();

    var zvm = try lib.ZVM.initWithAllocator(alloc);
    defer zvm.deinit();

    // std.debug.print("ZVM\n\t.home_dir={s}\n\t.base_dir={?}\n\t", .{ zvm.home_dir, zvm.base_dir });

    var index: i32 = 0;
    while (iter.next()) |arg| {
        if (lib.streql(arg, "install")) {
           if (iter.next()) |version| {
            try zvm.install(version);
           }
        }

        index += 1;
    }

    // // Prints to stderr (it's a shortcut based on `std.io.getStdErr()`)
    // std.debug.print("All your {s} are belong to us.\n", .{"codebase"});

    // // stdout is for the actual output of your application, for example if you
    // // are implementing gzip, then only the compressed bytes should be sent to
    // // stdout, not any debugging messages.
    // const stdout_file = std.io.getStdOut().writer();
    // var bw = std.io.bufferedWriter(stdout_file);
    // const stdout = bw.writer();

    // try stdout.print("Run `zig build test` to run the tests.\n", .{});

    // try bw.flush(); // don't forget to flush!
}

// test "simple test" {
//     var list = std.ArrayList(i32).init(std.testing.allocator);
//     defer list.deinit(); // try commenting this out and see if zig detects the memory leak!
//     try list.append(42);
//     try std.testing.expectEqual(@as(i32, 42), list.pop());
// }
