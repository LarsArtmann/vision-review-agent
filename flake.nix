{
  description = "Vision Review Agent - AI-powered screenshot and image analysis SDK";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        goVersion = "1.26.2";
        coverageThreshold = 70;

        # Build script with coverage check
        buildScript = pkgs.writeShellScriptBin "vision-build" ''
          set -euo pipefail
          echo "Building..."
          go build ./...
          echo "Building CLI..."
          go build -o vision-cli ./cmd/vision
        '';

        testScript = pkgs.writeShellScriptBin "vision-test" ''
          set -euo pipefail
          echo "Running tests with coverage threshold ${toString coverageThreshold}%..."
          go test ./... -v -coverprofile=coverage.out -coverpkg=./...
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Coverage: $coverage%"
          if (( $(echo "$coverage < ${toString coverageThreshold}" | bc -l) )); then
            echo "ERROR: Coverage $coverage% is below threshold ${toString coverageThreshold}%"
            exit 1
          fi
          echo "Coverage check passed!"
        '';

        testRaceScript = pkgs.writeShellScriptBin "vision-test-race" ''
          set -euo pipefail
          echo "Running tests with race detector..."
          go test ./... -race
        '';

        lintScript = pkgs.writeShellScriptBin "vision-lint" ''
          set -euo pipefail
          echo "Running go vet..."
          go vet ./...
          echo "Running golangci-lint..."
          golangci-lint run ./...
        '';

        fmtScript = pkgs.writeShellScriptBin "vision-fmt" ''
          set -euo pipefail
          echo "Formatting..."
          gofmt -w .
        '';

        cleanScript = pkgs.writeShellScriptBin "vision-clean" ''
          set -euo pipefail
          rm -f vision-cli coverage.out
          go clean ./...
        '';
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "vision-review-agent";
          version = "0.1.0";
          src = ./.;
          vendorHash = null;
          meta = with pkgs.lib; {
            description = "AI-powered screenshot and image analysis SDK";
            license = licenses.mit;
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go_1_26
            golangci-lint
            gopls
            gotools
            bc
            buildScript
            testScript
            testRaceScript
            lintScript
            fmtScript
            cleanScript
          ];
        };

        apps = {
          default = {
            type = "app";
            program = "${self.packages.${system}.default}/bin/vision";
          };
        };
      }) // {
        overlays.default = final: prev: {
          vision-review-agent = self.packages.${prev.system}.default;
        };
      };
}
