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

        # To enable package builds, run:
        #   nix build .#default
        # If vendorHash is wrong, copy the correct hash from the error output.
        # vision-review-agent = pkgs.buildGoModule {
        #   pname = "vision-review-agent";
        #   version = "0.1.0";
        #   src = ./.;
        #   vendorHash = "<run nix build to get hash>";
        #   meta = with pkgs.lib; {
        #     description = "AI-powered screenshot and image analysis SDK";
        #     license = licenses.mit;
        #   };
        # };
      in
      {
        # packages.default = vision-review-agent;

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go_1_26
            golangci-lint
            gopls
            gotools
            just
          ];

          shellHook = ''
            echo "Vision Review Agent dev shell"
            echo "  just         - list available commands"
            echo "  just test    - run tests"
            echo "  just coverage - run tests with coverage threshold"
            echo "  just lint    - run golangci-lint"
          '';
        };
      });
}
