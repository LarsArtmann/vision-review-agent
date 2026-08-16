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
          inherit (pkgs) lib;
          version = self.rev or self.dirtyRev or "dev";
          # Shared by every Go package in this flake: one source, one vendored
          # dependency set (extracted so dependency bumps touch exactly one line).
          src = lib.cleanSource ./.;
          vendorHash = "sha256-gwJZWwfWl/ldNGQJWyNIpkgpu7j9w6ZtoDlp6o2rC4k=";

          # NixOS-module evaluation fixtures for the checks below (enabled +
          # disabled). Evaluating forces the systemd units; ExecStart depends
          # on the visionreviewd store path, so the enabled case also proves
          # the package builds. The llama unit's ExecStart is deliberately NOT
          # forced: it would build llama-cpp (~heavy) just for an eval check;
          # the plain `description` literal proves the unit exists when
          # enabled.
          nixosModuleEnabled = lib.nixosSystem {
            system = pkgs.stdenv.hostPlatform.system;
            modules = [
              self.nixosModules.visionreviewd
              {
                services.vision-review-agent = {
                  enable = true;
                  configFile = "/etc/visionreviewd/config.json";
                  llamaServer.enable = true;
                };
              }
            ];
          };

          nixosModuleDisabled = lib.nixosSystem {
            system = pkgs.stdenv.hostPlatform.system;
            modules = [ self.nixosModules.visionreviewd ];
          };
        in
        {
          packages.default = pkgs.buildGoModule {
            pname = "vision-review-agent";
            inherit version src vendorHash;
            proxyVendor = true;
            # go-cqrs-lite imports encoding/json/v2, which only exists under
            # GOEXPERIMENT=jsonv2 (the local dev env sets it via `go env`).
            env.GOEXPERIMENT = "jsonv2";
            # Inject the nix-derived version into the Go binary so `vision
            # -version` reports the real release tag instead of the hardcoded
            # dev default in cmd/vision/main.go.
            ldflags = [ "-X main.version=${version}" ];
            meta = with lib; {
              description = "AI-powered screenshot and image analysis SDK";
              homepage = "https://github.com/LarsArtmann/vision-review-agent";
              # PROPRIETARY license — see LICENSE. The repo is source-available,
              # not open-source. `licenses.unfree` is the honest nixpkgs marker.
              license = licenses.unfree;
              platforms = platforms.unix ++ platforms.windows;
              maintainers = [
                {
                  name = "Lars Artmann";
                  github = "LarsArtmann";
                }
              ];
              mainProgram = "vision";
            };
          };

          packages.visionreviewd = pkgs.buildGoModule {
            pname = "visionreviewd";
            inherit version src vendorHash;
            proxyVendor = true;
            subPackages = [ "cmd/visionreviewd" ];
            env.GOEXPERIMENT = "jsonv2";
            # Inject the nix-derived version into the daemon binary so
            # `visionreviewd version` reports the real release tag instead of
            # the hardcoded default in cmd/visionreviewd/main.go.
            ldflags = [ "-X main.version=${version}" ];
            meta = with lib; {
              description = "Event-sourced UI review daemon over local vision models";
              homepage = "https://github.com/LarsArtmann/vision-review-agent";
              license = licenses.unfree;
              platforms = platforms.unix ++ platforms.windows;
              maintainers = [
                {
                  name = "Lars Artmann";
                  github = "LarsArtmann";
                }
              ];
              mainProgram = "visionreviewd";
            };
          };

          apps = {
            default = {
              type = "app";
              program = lib.getExe config.packages.default;
              meta.description = "Run the vision-review-agent CLI";
            };

            test = {
              type = "app";
              program = pkgs.writeShellApplication {
                name = "run-test";
                runtimeInputs = [ pkgs.go_1_26 ];
                text = "go test -race -v -coverprofile=coverage.out ./...";
              };
              meta.description = "Run the full Go test suite with race detector and coverage";
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
              meta.description = "Run golangci-lint over the whole module";
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
          }
          // lib.optionalAttrs pkgs.stdenv.hostPlatform.isLinux {
            # The daemon surface: build, version smoke, module eval.
            visionreviewd = config.packages.visionreviewd;

            # Catches the silent-empty-build class: if buildGoModule ever
            # "succeeds" with an empty output again (e.g. the GOEXPERIMENT
            # sandbox regression), running the binary fails here.
            visionreviewd-version-smoke = pkgs.runCommand "visionreviewd-version-smoke" { } ''
              version_output=$(${lib.getExe config.packages.visionreviewd} version)
              echo "$version_output" | grep -q '^visionreviewd .\+'
              echo "$version_output" > $out
            '';

            nixos-module-enabled =
              pkgs.runCommand "visionreviewd-nixos-module-enabled-eval"
                {
                  execStart = toString (
                    nixosModuleEnabled.config.systemd.services.visionreviewd.serviceConfig.ExecStart or ""
                  );
                  llamaUnit = lib.optionalString (
                    nixosModuleEnabled.config.systemd.services ? llama-vision-server
                  ) nixosModuleEnabled.config.systemd.services.llama-vision-server.description;
                }
                ''
                  [ -n "$execStart" ] || {
                    echo "visionreviewd service ExecStart evaluated empty"
                    exit 1
                  }
                  [ -n "$llamaUnit" ] || {
                    echo "llama-vision-server unit missing despite llamaServer.enable"
                    exit 1
                  }
                  printf 'module evaluates enabled: %s\n' "$execStart" > $out
                '';

            nixos-module-disabled =
              pkgs.runCommand "visionreviewd-nixos-module-disabled-eval"
                {
                  moduleEnable = nixosModuleDisabled.config.services.vision-review-agent.enable;
                  daemonUnits = builtins.attrNames nixosModuleDisabled.config.systemd.services;
                }
                ''
                  [ "$moduleEnable" = "false" ] || {
                    echo "services.vision-review-agent.enable must default to false"
                    exit 1
                  }
                  (printf '%s\n' "$daemonUnits" | grep -qx visionreviewd) && {
                    echo "visionreviewd unit must be absent when disabled"
                    exit 1
                  }
                  echo "module evaluates disabled (defaults)" > $out
                '';
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
        visionreviewd = self.packages.${final.stdenv.system}.visionreviewd;
      };

      flake.nixosModules.visionreviewd = import ./nixos/visionreviewd.nix self;
    };
}
