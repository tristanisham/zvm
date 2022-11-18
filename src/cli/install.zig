const std = @import("std");
const builtin = @import("builtin");

/// Fetches the user's home directory.
// Pulls in $HOME. More needed to determine if valid on Windows.
pub fn homeDir(alloc: std.mem.Allocator) ?[]u8 {
    return std.process.getEnvVarOwned(alloc, "HOME") catch null;
}