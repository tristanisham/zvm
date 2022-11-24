const std = @import("std");

pub const ZVMConfig = struct {
    default_ver: []u8,


    pub fn marshal(self: *ZVMConfig, alloc: *std.mem.Allocator) !*std.ArrayList(u8) {
        var string = std.ArrayList(u8).init(alloc.*);
        try std.json.stringify(self, .{}, string.writer());
        return &string;
    }  
};