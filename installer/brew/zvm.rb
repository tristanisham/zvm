class Zvm < Formula
  desc "Version manager for the Zig programming language"
  homepage "https://www.zvm.app"
  url "https://github.com/tristanisham/zvm/archive/refs/tags/v0.8.29.tar.gz"
  sha256 "d41875911b44bf0faf01322b6ec46958d73a80e49a8d50db4380c1f064ddc6cd"
  license "MIT"

  depends_on "go" => :build

  def install
    # std_go_args already supplies -s -w, -trimpath and -o=bin/zvm, so ldflags
    # must not repeat them. The single quotes matter: the message contains
    # spaces and Go re-splits the -ldflags string with quote handling.
    ldflags = "-X 'main.BuildUpgradeMessage=Use `brew upgrade zvm` to update ZVM.'"
    system "go", "build", *std_go_args(ldflags:, tags: "noAutoUpgrades")
  end

  def caveats
    <<~EOS
      zvm manages Zig toolchains in ~/.zvm. Add the active toolchain to your PATH:
        export PATH="$HOME/.zvm/bin:$PATH"

      Then install and select a Zig version:
        zvm i master && zvm use master
    EOS
  end

  test do
    # Homebrew sets HOME to testpath, so zvm operates on a throwaway ~/.zvm.
    assert_match "No local Zig installs", shell_output("#{bin}/zvm ls")
    assert_path_exists testpath/".zvm/settings.json"

    # Self-upgrade must stay compiled out so Homebrew owns version management.
    assert_match "built with noAutoUpgrades", shell_output("#{bin}/zvm upgrade")
  end
end
