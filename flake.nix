{
  description = "Vision Review Agent - AI-powered screenshot and image analysis SDK";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };

    systems.url = "github:nix-systems/default";

    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{
      self,
      flake-parts,
      ...
    }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import inputs.systems;
      imports = [ inputs.treefmt-nix.flakeModule ];

      perSystem =
        {
          config,
          system,
          ...
        }:
        let
          # This project is proprietary (see LICENSE). Import nixpkgs with
          # allowUnfree so `nix build` succeeds in pure evaluation without
          # needing --impure / NIXPKGS_ALLOW_UNFREE at the call site.
          pkgs = import inputs.nixpkgs {
            inherit system;
            config.allowUnfree = true;
          };
          lib = pkgs.lib;
          version = self.rev or self.dirtyRev or "dev";
        in
        {
          packages.default = pkgs.buildGoModule {
            pname = "vision-review-agent";
            inherit version;
            src = lib.cleanSource ./.;
            vendorHash = "sha256-vtWvYhyC1AYshWGcfBBh8osNUA6ekFJcY+OdIgqJiV4=";
            proxyVendor = true;
            meta = with lib; {
              description = "AI-powered screenshot and image analysis SDK";
              # PROPRIETARY license — see LICENSE. The repo is source-available,
              # not open-source. `licenses.unfree` is the honest nixpkgs marker.
              license = licenses.unfree;
              maintainers = [
                {
                  name = "Lars Artmann";
                  github = "LarsArtmann";
                }
              ];
              mainProgram = "vision-review-agent";
            };
          };

          apps = {
            default = {
              type = "app";
              program = lib.getExe config.packages.default;
            };

            test = {
              type = "app";
              program = pkgs.writeShellApplication {
                name = "run-test";
                runtimeInputs = [ pkgs.go_1_26 ];
                text = "go test -race -v -coverprofile=coverage.out ./...";
              };
            };

            lint = {
              type = "app";
              program = pkgs.writeShellApplication {
                name = "run-lint";
                runtimeInputs = [
                  pkgs.go_1_26
                  pkgs.golangci-lint
                ];
                text = "golangci-lint run ./...";
              };
            };
          };

          devShells = {
            default = pkgs.mkShell {
              packages = with pkgs; [
                go_1_26
                golangci-lint
                gopls
                gotools
              ];

              inputsFrom = [ config.packages.default ];

              GOWORK = "off";

              shellHook = ''
                echo "Vision Review Agent dev shell"
              '';
            };

            ci = pkgs.mkShellNoCC {
              packages = [
                pkgs.go_1_26
                pkgs.golangci-lint
              ];
              GOWORK = "off";
            };
          };

          checks = {
            format = config.treefmt.build.check self;
            build = config.packages.default;
            test = config.packages.default.overrideAttrs (_: {
              doCheck = true;
            });
          };

          treefmt = {
            projectRootFile = "go.mod";
            programs = {
              gofumpt.enable = true;
              goimports.enable = true;
              nixfmt.enable = true;
            };
          };
        };

      flake.overlays.default = final: _prev: {
        vision-review-agent = self.packages.${final.stdenv.system}.default;
      };
    };
}
