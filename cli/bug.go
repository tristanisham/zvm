// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/log"

	"github.com/tristanisham/zvm/cli/meta"
)

// bugReportEndpoint is GitHub's new-issue form for ZVM. A "body" query
// parameter prefills the issue text.
const bugReportEndpoint = "https://github.com/tristanisham/zvm/issues/new"

// bugReportEnvVars are the environment variables ZVM reads. They are reported
// whether or not they are set, so an empty value is evidence rather than an
// omission.
var bugReportEnvVars = []string{
	"ZVM_PATH",
	"ZVM_DEBUG",
	"ZVM_SET_CU",
	"ZVM_SKIP_TLS_VERIFY",
	"ZVM_TARGET_OS",
	"ZVM_TARGET_ARCH",
	"ZVM_HTTP_TIMEOUT",
	"ZVM_INSTALL",
}

// Bug opens a prefilled ZVM bug report in the user's browser. When printOnly
// is set, or no browser can be launched, the URL is printed instead so the
// user can open it themselves.
func (z ZVM) Bug(printOnly bool) error {
	link := BugReportURL(z.BugReportBody())

	if !printOnly {
		fmt.Println("Please file a new issue at https://github.com/tristanisham/zvm/issues/new")
		if openBrowser(link) {
			return nil
		}
		fmt.Println("Opening a browser failed. Please file a report at:")
	}

	fmt.Println(link)
	return nil
}

// BugReportURL builds the GitHub new-issue URL that prefills body.
func BugReportURL(body string) string {
	return bugReportEndpoint + "?body=" + url.QueryEscape(body)
}

// BugReportBody renders the issue template ZVM prefills for `zvm bug`,
// including the environment details a maintainer needs to reproduce a report.
func (z ZVM) BugReportBody() string {
	var b strings.Builder

	b.WriteString("<!-- Please answer these questions before submitting your issue. Thanks! -->\n\n")

	b.WriteString("### What version of ZVM are you using (`zvm version`)?\n\n")
	b.WriteString("<pre>\n$ zvm version\nzvm version ")
	b.WriteString(meta.VerCopy)
	b.WriteString("\n</pre>\n\n")

	b.WriteString("### Does this issue reproduce with the latest release?\n\n\n")

	b.WriteString("### What operating system and processor architecture are you using?\n\n")
	b.WriteString("<details><summary><code>zvm bug</code> Output</summary><br><pre>\n")
	b.WriteString(z.bugEnvironment())
	b.WriteString("</pre></details>\n\n")

	b.WriteString("### What did you do?\n\n")
	b.WriteString("<!--\nIf possible, provide a recipe for reproducing the error.\n")
	b.WriteString("The exact `zvm` commands you ran, in order, is best.\n-->\n\n\n\n")

	b.WriteString("### What did you see happen?\n\n\n\n")

	b.WriteString("### What did you expect to see?\n\n")

	return b.String()
}

// bugEnvironment reports ZVM's configuration, the state of the installations
// it manages, and the host it is running on.
func (z ZVM) bugEnvironment() string {
	var b strings.Builder

	write := func(key, value string) {
		fmt.Fprintf(&b, "%s='%s'\n", key, value)
	}

	write("ZVM_VERSION", meta.VERSION)
	write("GOOS", runtime.GOOS)
	write("GOARCH", runtime.GOARCH)
	write("GOVERSION", runtime.Version())

	for _, key := range bugReportEnvVars {
		// ZVM_PATH is reported as the directory actually in use, so a report
		// from an unset environment still shows where ZVM looked.
		if key == "ZVM_PATH" {
			write(key, redactPath(z.baseDir))
			continue
		}
		write(key, redactPath(os.Getenv(key)))
	}

	write("MIRROR_LIST_URL", redactSetting(z.Settings.MirrorListUrl))
	write("VERSION_MAP_URL", redactSetting(z.Settings.VersionMapUrl))
	write("ZLS_VERSION_MAP_URL", redactSetting(z.Settings.ZlsVMU))
	write("MINISIGN_PUB_KEY", redactSetting(z.Settings.MinisignPubKey))
	write("USE_COLOR", fmt.Sprint(z.Settings.UseColor))
	write("ALWAYS_FORCE_INSTALL", fmt.Sprint(z.Settings.AlwaysForceInstall))

	installed, err := z.GetInstalledVersions()
	if err != nil {
		log.Debug("bug: listing installed versions", "error", err)
	}
	write("INSTALLED_ZIG", strings.Join(installed, " "))

	write("ACTIVE_ZIG", z.bugBinaryVersion("zig", "version"))
	write("ACTIVE_ZLS", z.bugBinaryVersion("zls", "--version"))

	b.WriteString(bugOSDetails())

	return b.String()
}

// bugBinaryVersion reports the version of a binary ZVM currently links into
// its bin directory. A missing or broken link reports an empty version rather
// than failing the report.
func (z ZVM) bugBinaryVersion(name string, versionArg string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	out, err := exec.Command(filepath.Join(z.baseDir, "bin", name), versionArg).Output()
	if err != nil {
		log.Debug("bug: reading binary version", "binary", name, "error", err)
		return ""
	}

	return strings.TrimSpace(string(out))
}

// bugCommandOutput runs a host-inspection command for the environment dump.
// These commands are best-effort: anything that fails contributes nothing.
func bugCommandOutput(label string, name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		log.Debug("bug: running host command", "command", name, "error", err)
		return ""
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return ""
	}

	return label + ": " + trimmed + "\n"
}

// redactPath rewrites a path inside the user's home directory to start with
// "~". The shape of the path is what matters when diagnosing a report; the
// account name it sits under is not, and it identifies the reporter.
func redactPath(path string) string {
	home, err := os.UserHomeDir()
	if path == "" || err != nil || home == "" || home == string(filepath.Separator) {
		return path
	}

	if path == home {
		return "~"
	}

	if rest, found := strings.CutPrefix(path, home+string(filepath.Separator)); found {
		return "~" + string(filepath.Separator) + rest
	}

	return path
}

// redactSetting reports a configured value that may point somewhere private.
// Values ZVM ships or documents describe no one, so they are reported as they
// are; anything else the user chose is reported as a digest.
func redactSetting(value string) string {
	switch value {
	case "", "disabled",
		DefaultSettings.MirrorListUrl,
		DefaultSettings.VersionMapUrl,
		DefaultSettings.ZlsVMU,
		DefaultSettings.MinisignPubKey,
		MachVersionMapUrl:
		return value
	}

	return "custom " + shortDigest(value)
}

// shortDigest tags a value so that two reports carrying the same one can be
// recognized as the same.
//
// It is a correlation tag, not anonymization. A URL or path is drawn from a
// small set of likely candidates, so anyone willing to hash their guesses can
// recover the original. It is here to answer "is this the same mirror as the
// last report?" without reproducing a URL the reporter may not want public --
// if the value itself matters, ask them for it.
func shortDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:6])
}
