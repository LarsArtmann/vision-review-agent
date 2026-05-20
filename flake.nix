{
  description = "Vision Review Agent - AI-powered screenshot and image analysis SDK";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      perSystem = flake-utils.lib.eachDefaultSystem (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          version = self.rev or self.dirtyRev or "dev";
        in
        {
          packages.default = pkgs.buildGoModule {
            pname = "vision-review-agent";
            inherit version;
            src = ./.;
            vendorHash = "sha256-XOYhWymnhQZNSdlWVFE8MsJdote7HtPA53MqIsDWZ7s=";
            meta = with pkgs.lib; {
              description = "AI-powered screenshot and image analysis SDK";
              license = licenses.mit;
              mainProgram = "vision-review-agent";
            };
          };

          devShells.default = pkgs.mkShell {
            packages = with pkgs; [
              go_1_26
              golangci-lint
              gopls
              gotools
              just
            ];

            GOWORK = "off";

            shellHook = ''
              echo "Vision Review Agent dev shell"
              echo "  just         - list available commands"
              echo "  just test    - run tests"
              echo "  just coverage - run tests with coverage threshold"
              echo "  just lint    - run golangci-lint"
            '';
          };

          checks.build = self.packages.${system}.default;

          formatter = pkgs.nixfmt;
        });
    in
    perSystem // {
      overlays.default = final: prev: {
        vision-review-agent = self.packages.${final.stdenv.system}.default;
      };
    };
}
