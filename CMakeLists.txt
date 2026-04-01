cmake_minimum_required(VERSION 3.20)
project(zvm-windows-installer LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# ── Only build on Windows ────────────────────────────────────────────────
if(NOT WIN32)
    message(STATUS "[SKIP] zvm-windows-installer — not a Windows target, nothing to build")
    return()
endif()

message(STATUS "[BUILD] zvm-windows-installer.exe — native Windows installer for WinGet")
message(STATUS "[SKIP]  zvm (Go binary) — not managed by this CMake project")

# ── Installer executable ─────────────────────────────────────────────────
add_executable(zvm-windows-installer install.cpp)

target_compile_definitions(zvm-windows-installer PRIVATE
    UNICODE _UNICODE _CRT_SECURE_NO_WARNINGS
)

# Release-mode defaults when not explicitly set
if(NOT CMAKE_BUILD_TYPE AND NOT CMAKE_CONFIGURATION_TYPES)
    set(CMAKE_BUILD_TYPE Release CACHE STRING "Build type" FORCE)
    message(STATUS "Defaulting to Release build")
endif()

# MSVC flags
if(MSVC)
    target_compile_options(zvm-windows-installer PRIVATE /W4 /O2 /EHsc)
    target_link_options(zvm-windows-installer PRIVATE /SUBSYSTEM:CONSOLE)
endif()

# MinGW / GCC / Clang flags
if(CMAKE_CXX_COMPILER_ID MATCHES "GNU|Clang" AND MINGW)
    target_compile_options(zvm-windows-installer PRIVATE -Wall -Wextra -O2)
    target_link_options(zvm-windows-installer PRIVATE -municode)
endif()

# Native Windows libraries (pragma comment(lib) covers MSVC; explicit for MinGW)
target_link_libraries(zvm-windows-installer PRIVATE
    winhttp ole32 oleaut32 shell32 shlwapi advapi32 user32
)

# Place binary in ./build
set_target_properties(zvm-windows-installer PROPERTIES
    RUNTIME_OUTPUT_DIRECTORY "${CMAKE_SOURCE_DIR}/build"
    RUNTIME_OUTPUT_DIRECTORY_RELEASE "${CMAKE_SOURCE_DIR}/build"
    RUNTIME_OUTPUT_DIRECTORY_DEBUG "${CMAKE_SOURCE_DIR}/build"
    OUTPUT_NAME "zvm-windows-installer"
)
