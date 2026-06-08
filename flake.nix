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
          version = "0.1.0";
          defaultPkg = pkgs.buildGoModule {
            pname = "vision-review-agent";
            inherit version;
            src = ./.;
            vendorHash = "sha256-XOYhWymnhQZNSdlWVFE8MsJdote7HtPA53MqIsDWZ7s=";
            proxyVendor = true;
            meta = with pkgs.lib; {
              description = "AI-powered screenshot and image analysis SDK";
              license = licenses.mit;
              mainProgram = "vision-review-agent";
            };
          };
        in
        {
          packages.default = defaultPkg;

          apps = {
            default = {
              type = "app";
              program = pkgs.lib.getExe defaultPkg;
            };

            test = {
              type = "app";
              program = "${pkgs.lib.getExe (pkgs.writeShellApplication {
                name = "run-test";
                runtimeInputs = [ pkgs.go_1_26 ];
                text = "go test -race -v -coverprofile=coverage.out ./...";
              })}";
            };

            lint = {
              type = "app";
              program = "${pkgs.lib.getExe (pkgs.writeShellApplication {
                name = "run-lint";
                runtimeInputs = [ pkgs.go_1_26 pkgs.golangci-lint ];
                text = "golangci-lint run ./...";
              })}";
            };
          };

          devShells.default = pkgs.mkShell {
            packages = with pkgs; [
              go_1_26
              golangci-lint
              gopls
              gotools
            ];

            GOWORK = "off";

            shellHook = ''
              echo "Vision Review Agent dev shell"
            '';
          };

          devShells.ci = pkgs.mkShellNoCC {
            packages = [ pkgs.go_1_26 pkgs.golangci-lint ];
            GOWORK = "off";
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
