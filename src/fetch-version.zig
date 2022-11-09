const std = @import("std");
const cURL = @cImport({
    @cInclude("curl/curl.h");
});

const CurlError = error{ CURLGlobalInitFailed, CURLHandleInitFailed, CouldNotSetURL, CouldNotSetWriteCallback, FailedToPerformRequest, CouldNotSetUserAgent };

pub fn fetchVersionJSON(response_buffer: *std.ArrayList(u8)) CurlError!void {

    // global curl init, or fail
    if (cURL.curl_global_init(cURL.CURL_GLOBAL_ALL) != cURL.CURLE_OK)
        return CurlError.CURLGlobalInitFailed;
    defer cURL.curl_global_cleanup();

    // curl easy handle init, or fail
    const handle = cURL.curl_easy_init() orelse return CurlError.CURLHandleInitFailed;
    defer cURL.curl_easy_cleanup(handle);

    // setup curl options
    if (cURL.curl_easy_setopt(handle, cURL.CURLOPT_URL, "https://ziglang.org/download/index.json") != cURL.CURLE_OK)
        return CurlError.CouldNotSetURL;

    if (cURL.curl_easy_setopt(handle, cURL.CURLOPT_USERAGENT, "zvm (Zig Version Manager)/v0.0.1") != cURL.CURLE_OK) {
        return CurlError.CouldNotSetUserAgent;
    }
    // set write function callbacks
    if (cURL.curl_easy_setopt(handle, cURL.CURLOPT_WRITEFUNCTION, writeToArrayListCallback) != cURL.CURLE_OK) {
        return CurlError.CouldNotSetWriteCallback;
    }
    if (cURL.curl_easy_setopt(handle, cURL.CURLOPT_WRITEDATA, response_buffer) != cURL.CURLE_OK) {
        return CurlError.CouldNotSetWriteCallback;
    }

    // perform
    if (cURL.curl_easy_perform(handle) != cURL.CURLE_OK) {
        return CurlError.FailedToPerformRequest;
    }
}

fn writeToArrayListCallback(data: *anyopaque, size: c_uint, nmemb: c_uint, user_data: *anyopaque) callconv(.C) c_uint {
    var buffer = @intToPtr(*std.ArrayList(u8), @ptrToInt(user_data));
    var typed_data = @intToPtr([*]u8, @ptrToInt(data));
    buffer.appendSlice(typed_data[0 .. nmemb * size]) catch return 0;
    return nmemb * size;
}

/// parseVersionJSON takes the resturned result from fetchVersionJSON and parses it into a std.json.ValueTree.
pub fn parseVersionJSON(json: *std.ArrayList(u8), alloc: *std.heap.ArenaAllocator) !std.json.ValueTree {
    // std.debug.print("{s}", .{json.items});
    var parser = std.json.Parser.init(alloc.*.allocator(), false);

    var tree = try parser.parse(json.items);
    return tree;
}

pub fn downloadFile(url: []const u8, path: []const u8, alloc: *std.heap.ArenaAllocator) CurlError!std.fs.File.WriteError!void {
    var buf = std.ArrayList(u8).init(alloc.*.allocator());
    defer buf.deinit();

    if (cURL.curl_global_init(cURL.CURL_GLOBAL_ALL) != cURL.CURLE_OK) {
        return CurlError.CURLGlobalInitFailed;
    }
    defer cURL.curl_global_cleanup();

    // curl easy handle init, or fail
    const handle = cURL.curl_easy_init() orelse return CurlError.CURLHandleInitFailed;
    defer cURL.curl_easy_cleanup(handle);

    // setup curl options
    if (cURL.curl_easy_setopt(handle, cURL.CURLOPT_URL, url) != cURL.CURLE_OK)
        return CurlError.CouldNotSetURL;

    if (cURL.curl_easy_setopt(handle, cURL.CURLOPT_USERAGENT, "zvm (Zig Version Manager)/v0.0.1") != cURL.CURLE_OK) {
        return CurlError.CouldNotSetUserAgent;
    }
    // set write function callbacks
    if (cURL.curl_easy_setopt(handle, cURL.CURLOPT_WRITEFUNCTION, writeToArrayListCallback) != cURL.CURLE_OK) {
        return CurlError.CouldNotSetWriteCallback;
    }
    if (cURL.curl_easy_setopt(handle, cURL.CURLOPT_WRITEDATA, buf) != cURL.CURLE_OK) {
        return CurlError.CouldNotSetWriteCallback;
    }

    if (cURL.curl_easy_perform(handle) != cURL.CURLE_OK) {
        return CurlError.FailedToPerformRequest;
    }

    const file = try std.fs.createFileAbsolute(path, .{ .read = true });
    defer file.close();
    try file.writeAll(buf.items);
}
