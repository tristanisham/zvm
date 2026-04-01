// install.cpp — Native Windows installer for ZVM
// Compile (MSVC):  cl install.cpp
// Compile (MinGW): g++ -municode -O2 install.cpp -lwinhttp -lole32 -loleaut32 -lshell32 -lshlwapi -ladvapi32 -luser32 -o install.exe
//
// Uses only native Windows APIs:
//   WinHTTP   — HTTPS download with automatic redirect following
//   Shell COM — zip extraction via IShellDispatch / Folder::CopyHere
//   Registry  — persistent PATH and ZVM_INSTALL setup
//   Win32     — file operations, process creation, architecture detection

#ifndef _CRT_SECURE_NO_WARNINGS
#define _CRT_SECURE_NO_WARNINGS
#endif

#ifndef UNICODE
#define UNICODE
#endif
#ifndef _UNICODE
#define _UNICODE
#endif

#include <windows.h>
#include <winhttp.h>
#include <shlobj.h>
#include <shldisp.h>
#include <shlwapi.h>

#include <cstdio>
#include <string>

#pragma comment(lib, "winhttp.lib")
#pragma comment(lib, "ole32.lib")
#pragma comment(lib, "oleaut32.lib")
#pragma comment(lib, "shell32.lib")
#pragma comment(lib, "shlwapi.lib")
#pragma comment(lib, "advapi32.lib")
#pragma comment(lib, "user32.lib")

// ── Helpers ─────────────────────────────────────────────────────────────

[[noreturn]] static void Fatal(const wchar_t* msg) {
    fwprintf(stderr, L"Install Failed - %s\n", msg);
    ExitProcess(1);
}

static void EnsureDir(const std::wstring& path) {
    SHCreateDirectoryExW(nullptr, path.c_str(), nullptr);
}

static bool FileExists(const std::wstring& path) {
    DWORD attr = GetFileAttributesW(path.c_str());
    return attr != INVALID_FILE_ATTRIBUTES && !(attr & FILE_ATTRIBUTE_DIRECTORY);
}

static void DeleteTree(const std::wstring& path) {
    // SHFileOperation requires double-null terminated string
    std::wstring from = path;
    from.push_back(L'\0');
    SHFILEOPSTRUCTW op = {};
    op.wFunc  = FO_DELETE;
    op.pFrom  = from.c_str();
    op.fFlags = FOF_NOCONFIRMATION | FOF_NOERRORUI | FOF_SILENT;
    SHFileOperationW(&op);
}

static void PrintGreen(const wchar_t* text) {
    HANDLE h = GetStdHandle(STD_OUTPUT_HANDLE);
    CONSOLE_SCREEN_BUFFER_INFO csbi;
    GetConsoleScreenBufferInfo(h, &csbi);
    SetConsoleTextAttribute(h, FOREGROUND_GREEN | FOREGROUND_INTENSITY);
    wprintf(L"%s", text);
    SetConsoleTextAttribute(h, csbi.wAttributes);
}

// ── Download via WinHTTP ────────────────────────────────────────────────

static bool DownloadFile(const std::wstring& url, const std::wstring& outPath) {
    URL_COMPONENTSW uc = {};
    uc.dwStructSize = sizeof(uc);
    wchar_t host[256] = {}, path[2048] = {};
    uc.lpszHostName    = host;
    uc.dwHostNameLength = _countof(host);
    uc.lpszUrlPath     = path;
    uc.dwUrlPathLength = _countof(path);

    if (!WinHttpCrackUrl(url.c_str(), 0, 0, &uc))
        return false;

    HINTERNET hSession = WinHttpOpen(L"ZVM-Installer/1.0",
        WINHTTP_ACCESS_TYPE_DEFAULT_PROXY,
        WINHTTP_NO_PROXY_NAME, WINHTTP_NO_PROXY_BYPASS, 0);
    if (!hSession) return false;

    DWORD tls = WINHTTP_FLAG_SECURE_PROTOCOL_TLS1_2;
    WinHttpSetOption(hSession, WINHTTP_OPTION_SECURE_PROTOCOLS, &tls, sizeof(tls));

    HINTERNET hConn = WinHttpConnect(hSession, host, uc.nPort, 0);
    if (!hConn) { WinHttpCloseHandle(hSession); return false; }

    DWORD reqFlags = (uc.nScheme == INTERNET_SCHEME_HTTPS) ? WINHTTP_FLAG_SECURE : 0;
    HINTERNET hReq = WinHttpOpenRequest(hConn, L"GET", path,
        nullptr, WINHTTP_NO_REFERER, WINHTTP_DEFAULT_ACCEPT_TYPES, reqFlags);
    if (!hReq) { WinHttpCloseHandle(hConn); WinHttpCloseHandle(hSession); return false; }

    DWORD redirectPolicy = WINHTTP_OPTION_REDIRECT_POLICY_ALWAYS;
    WinHttpSetOption(hReq, WINHTTP_OPTION_REDIRECT_POLICY, &redirectPolicy, sizeof(redirectPolicy));

    if (!WinHttpSendRequest(hReq, nullptr, 0, nullptr, 0, 0, 0) ||
        !WinHttpReceiveResponse(hReq, nullptr)) {
        WinHttpCloseHandle(hReq); WinHttpCloseHandle(hConn); WinHttpCloseHandle(hSession);
        return false;
    }

    DWORD status = 0, sz = sizeof(status);
    WinHttpQueryHeaders(hReq,
        WINHTTP_QUERY_STATUS_CODE | WINHTTP_QUERY_FLAG_NUMBER,
        WINHTTP_HEADER_NAME_BY_INDEX, &status, &sz, WINHTTP_NO_HEADER_INDEX);
    if (status != 200) {
        WinHttpCloseHandle(hReq); WinHttpCloseHandle(hConn); WinHttpCloseHandle(hSession);
        return false;
    }

    HANDLE hFile = CreateFileW(outPath.c_str(), GENERIC_WRITE, 0, nullptr,
        CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, nullptr);
    if (hFile == INVALID_HANDLE_VALUE) {
        WinHttpCloseHandle(hReq); WinHttpCloseHandle(hConn); WinHttpCloseHandle(hSession);
        return false;
    }

    DWORD contentLen = 0;
    sz = sizeof(contentLen);
    WinHttpQueryHeaders(hReq,
        WINHTTP_QUERY_CONTENT_LENGTH | WINHTTP_QUERY_FLAG_NUMBER,
        WINHTTP_HEADER_NAME_BY_INDEX, &contentLen, &sz, WINHTTP_NO_HEADER_INDEX);

    BYTE buf[65536];
    DWORD totalRead = 0, bytesRead;
    while (WinHttpReadData(hReq, buf, sizeof(buf), &bytesRead) && bytesRead > 0) {
        DWORD written;
        WriteFile(hFile, buf, bytesRead, &written, nullptr);
        totalRead += bytesRead;
        if (contentLen > 0)
            wprintf(L"\r  Downloading... %u%%", (unsigned)(100ULL * totalRead / contentLen));
    }
    if (contentLen > 0)
        wprintf(L"\r  Downloading... done.    \n");

    CloseHandle(hFile);
    WinHttpCloseHandle(hReq);
    WinHttpCloseHandle(hConn);
    WinHttpCloseHandle(hSession);
    return totalRead > 0;
}

// ── Zip extraction via Shell COM ────────────────────────────────────────

static bool ExtractZip(const std::wstring& zipPath, const std::wstring& destDir) {
    CLSID clsid;
    if (FAILED(CLSIDFromProgID(L"Shell.Application", &clsid)))
        return false;

    IShellDispatch* pSD = nullptr;
    if (FAILED(CoCreateInstance(clsid, nullptr, CLSCTX_INPROC_SERVER,
            __uuidof(IShellDispatch), (void**)&pSD)) || !pSD)
        return false;

    auto makeVariantBstr = [](const std::wstring& s) -> VARIANT {
        VARIANT v = {};
        v.vt = VT_BSTR;
        v.bstrVal = SysAllocString(s.c_str());
        return v;
    };

    VARIANT vSrc = makeVariantBstr(zipPath);
    Folder* pSrc = nullptr;
    HRESULT hr = pSD->NameSpace(vSrc, &pSrc);
    SysFreeString(vSrc.bstrVal);
    if (FAILED(hr) || !pSrc) { pSD->Release(); return false; }

    VARIANT vDst = makeVariantBstr(destDir);
    Folder* pDst = nullptr;
    hr = pSD->NameSpace(vDst, &pDst);
    SysFreeString(vDst.bstrVal);
    if (FAILED(hr) || !pDst) { pSrc->Release(); pSD->Release(); return false; }

    FolderItems* pItems = nullptr;
    pSrc->Items(&pItems);
    if (!pItems) { pDst->Release(); pSrc->Release(); pSD->Release(); return false; }

    VARIANT vItems = {};
    vItems.vt = VT_DISPATCH;
    vItems.pdispVal = pItems;

    VARIANT vOpts = {};
    vOpts.vt = VT_I4;
    vOpts.lVal = 0x0614;  // FOF_SILENT | FOF_NOCONFIRMATION | FOF_NOERRORUI | FOF_NOCONFIRMMKDIR

    hr = pDst->CopyHere(vItems, vOpts);

    pItems->Release();
    pDst->Release();
    pSrc->Release();
    pSD->Release();
    return SUCCEEDED(hr);
}

// ── Environment variables ───────────────────────────────────────────────

static std::wstring GetUserEnv(const wchar_t* name) {
    HKEY hKey;
    if (RegOpenKeyExW(HKEY_CURRENT_USER, L"Environment", 0,
            KEY_QUERY_VALUE, &hKey) != ERROR_SUCCESS)
        return {};

    DWORD type = 0, size = 0;
    RegQueryValueExW(hKey, name, nullptr, &type, nullptr, &size);
    if (size == 0) { RegCloseKey(hKey); return {}; }

    std::wstring val(size / sizeof(wchar_t), L'\0');
    RegQueryValueExW(hKey, name, nullptr, &type, (BYTE*)val.data(), &size);
    RegCloseKey(hKey);

    while (!val.empty() && val.back() == L'\0') val.pop_back();
    return val;
}

static void SetUserEnv(const wchar_t* name, const std::wstring& value) {
    HKEY hKey;
    if (RegOpenKeyExW(HKEY_CURRENT_USER, L"Environment", 0,
            KEY_SET_VALUE, &hKey) != ERROR_SUCCESS)
        return;

    DWORD type = (_wcsicmp(name, L"Path") == 0) ? REG_EXPAND_SZ : REG_SZ;
    RegSetValueExW(hKey, name, 0, type,
        (const BYTE*)value.c_str(), (DWORD)((value.size() + 1) * sizeof(wchar_t)));
    RegCloseKey(hKey);
}

static void BroadcastEnvChange() {
    DWORD_PTR result;
    SendMessageTimeoutW(HWND_BROADCAST, WM_SETTINGCHANGE, 0,
        (LPARAM)L"Environment", SMTO_ABORTIFHUNG, 5000, &result);
}

static bool PathContains(const std::wstring& pathStr, const std::wstring& dir) {
    std::wstring lp = pathStr, ld = dir;
    for (auto& c : lp) c = (wchar_t)towlower(c);
    for (auto& c : ld) c = (wchar_t)towlower(c);
    while (!ld.empty() && ld.back() == L'\\') ld.pop_back();

    size_t pos = 0;
    while (pos <= lp.size()) {
        size_t end = lp.find(L';', pos);
        if (end == std::wstring::npos) end = lp.size();
        std::wstring seg = lp.substr(pos, end - pos);
        while (!seg.empty() && seg.back() == L'\\') seg.pop_back();
        if (seg == ld) return true;
        pos = end + 1;
    }
    return false;
}

static void AddToUserPath(const std::wstring& dir) {
    std::wstring path = GetUserEnv(L"Path");
    if (PathContains(path, dir)) return;
    if (!path.empty() && path.back() != L';') path += L';';
    path += dir;
    SetUserEnv(L"Path", path);
}

// ── Process verification ────────────────────────────────────────────────

static DWORD RunAndGetExitCode(const std::wstring& exe) {
    STARTUPINFOW si = {};
    si.cb = sizeof(si);
    si.dwFlags = STARTF_USESHOWWINDOW;
    si.wShowWindow = SW_HIDE;

    PROCESS_INFORMATION pi = {};
    std::wstring cmd = L"\"" + exe + L"\"";
    wchar_t cmdBuf[MAX_PATH + 4];
    wcsncpy(cmdBuf, cmd.c_str(), _countof(cmdBuf) - 1);
    cmdBuf[_countof(cmdBuf) - 1] = L'\0';

    if (!CreateProcessW(nullptr, cmdBuf, nullptr, nullptr, FALSE,
            CREATE_NO_WINDOW, nullptr, nullptr, &si, &pi))
        return (DWORD)-1;

    WaitForSingleObject(pi.hProcess, 10000);
    DWORD code = 0;
    GetExitCodeProcess(pi.hProcess, &code);
    CloseHandle(pi.hProcess);
    CloseHandle(pi.hThread);
    return code;
}

// ── Entry point ─────────────────────────────────────────────────────────

int wmain(int argc, wchar_t* argv[]) {
    // ── Architecture ────────────────────────────────────────────────────
    SYSTEM_INFO si;
    GetNativeSystemInfo(&si);

    const wchar_t* arch;
    switch (si.wProcessorArchitecture) {
    case PROCESSOR_ARCHITECTURE_AMD64: arch = L"amd64"; break;
    case PROCESSOR_ARCHITECTURE_ARM64: arch = L"arm64"; break;
    default:
        Fatal(L"ZVM requires a 64-bit environment.\n"
              L"Please ensure your system is 64-bit.");
    }

    // ── Flags ───────────────────────────────────────────────────────────
    bool noEnv = false;
    for (int i = 1; i < argc; i++)
        if (wcscmp(argv[i], L"--no-env") == 0) noEnv = true;

    // ── Paths ───────────────────────────────────────────────────────────
    wchar_t home[MAX_PATH];
    if (!GetEnvironmentVariableW(L"USERPROFILE", home, MAX_PATH))
        Fatal(L"Could not determine home directory (USERPROFILE not set).");

    std::wstring zvmRoot = std::wstring(home) + L"\\.zvm";
    std::wstring zvmSelf = zvmRoot + L"\\self";
    std::wstring zvmBin  = zvmRoot + L"\\bin";
    EnsureDir(zvmSelf);
    EnsureDir(zvmBin);

    std::wstring target  = std::wstring(L"zvm-windows-") + arch + L".zip";
    std::wstring url     = L"https://github.com/tristanisham/zvm/releases/latest/download/" + target;
    std::wstring zipPath = zvmSelf + L"\\" + target;

    // Remove stale zip
    DeleteFileW(zipPath.c_str());

    // ── COM init (STA required for Shell namespace) ─────────────────────
    CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED);

    // ── Download ────────────────────────────────────────────────────────
    wprintf(L"Downloading %s ...\n", target.c_str());
    if (!DownloadFile(url, zipPath)) {
        wprintf(L"URL: %s\n", url.c_str());
        Fatal(L"Could not download the release archive.");
    }
    if (!FileExists(zipPath))
        Fatal(L"Downloaded file does not exist. Did an antivirus delete it?");

    // ── Extract ─────────────────────────────────────────────────────────
    wprintf(L"Extracting...\n");
    if (!ExtractZip(zipPath, zvmSelf))
        Fatal(L"Could not extract the archive.");

    // CopyHere is asynchronous — pump messages and poll for zvm.exe
    std::wstring unzipped = std::wstring(L"zvm-windows-") + arch;
    std::wstring exeFlat  = zvmSelf + L"\\zvm.exe";
    std::wstring exeSub   = zvmSelf + L"\\" + unzipped + L"\\zvm.exe";

    bool found = false;
    for (int i = 0; i < 300 && !found; i++) {   // up to 30 s
        MSG msg;
        while (PeekMessageW(&msg, nullptr, 0, 0, PM_REMOVE)) {
            TranslateMessage(&msg);
            DispatchMessage(&msg);
        }
        found = FileExists(exeFlat) || FileExists(exeSub);
        if (!found) Sleep(100);
    }
    if (!found)
        Fatal(L"Extraction failed — zvm.exe not found. Archive may be corrupt or blocked by antivirus.");

    // If extracted into a subdirectory, move up
    if (FileExists(exeSub)) {
        DeleteFileW(exeFlat.c_str());
        MoveFileW(exeSub.c_str(), exeFlat.c_str());
        // Clean up the now-empty subdirectory
        DeleteTree(zvmSelf + L"\\" + unzipped);
    }

    // Clean up zip
    DeleteFileW(zipPath.c_str());

    // ── Verify ──────────────────────────────────────────────────────────
    DWORD exitCode = RunAndGetExitCode(exeFlat);
    if (exitCode == 0xC000001D) // STATUS_ILLEGAL_INSTRUCTION
        Fatal(L"zvm.exe is not compatible with your CPU.");
    if (exitCode == (DWORD)-1)
        Fatal(L"Could not launch zvm.exe to verify the installation.");

    // ── Success ─────────────────────────────────────────────────────────
    PrintGreen(L"ZVM");
    wprintf(L" was installed successfully!\n");
    wprintf(L"The binary is located at %s\\zvm.exe\n\n", zvmSelf.c_str());

    // ── Environment ─────────────────────────────────────────────────────
    if (!noEnv) {
        if (GetUserEnv(L"ZVM_INSTALL").empty())
            SetUserEnv(L"ZVM_INSTALL", zvmSelf);

        AddToUserPath(zvmSelf);
        AddToUserPath(zvmBin);
        BroadcastEnvChange();
    } else {
        wprintf(L"Skipping environment variable setup due to --no-env flag.\n\n");
    }

    wprintf(L"To get started, restart your terminal/editor, then type \"zvm\"\n\n");

    CoUninitialize();
    return 0;
}
