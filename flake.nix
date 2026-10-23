{
  description = "Zig Version Manager";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";

  outputs = { self, nixpkgs, ... }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.buildGoModule {
            pname = "zvm";
            version = "0.9.3";

            src = ./.;
            vendorHash = "sha256-mCLgyJRBT5ywqkm+xd9jT+DZWKrIj1vN+eIed2dto8c=";
            subPackages = [ "." ];

            tags = [ "noAutoUpgrades" ];

            # The sandbox sets HOME=/homeless-shelter, which is not writable,
            # and the main package creates ~/.zvm when its tests load.
            preCheck = ''
              export HOME=$(mktemp -d)
            '';

            ldflags = [
              "-s"
              "-w"
              "-X 'main.BuildUpgradeMessage=Use nix flake update and rebuild to upgrade ZVM.'"
            ];

            nativeBuildInputs = [ pkgs.installShellFiles ];
            postInstall = ''
              installManPage man/*.1
            '';

            meta = {
              description = "Zig Version Manager";
              homepage = "https://www.zvm.app";
              license = nixpkgs.lib.licenses.mit;
              mainProgram = "zvm";
            };
          };
        });

      apps = forAllSystems (system: {
        default = {
          type = "app";
          program = "${nixpkgs.lib.getExe self.packages.${system}.default}";
        };
      });

      devShells = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            packages = [ pkgs.go_1_26 ];
          };
        });

      formatter = forAllSystems (system: nixpkgs.legacyPackages.${system}.nixfmt);
    };
}
