# Activating visionreviewd on SystemNix

The daemon ships as `flake.nixosModules.visionreviewd` in this repo. SystemNix
wraps it in `modules/nixos/services/visionreviewd.nix`, which auto-registers
as `nixosModules.visionreviewd` and defaults the llama-server port to the
central registry entry `visionreviewd-llama` (8390, verified free).

The wrapper is deliberately **lazy**: it imports the upstream module only
when the locked revision of `vision-review-agent` actually ships
`nixosModules.visionreviewd`. Until then (and after any rollback) SystemNix
keeps evaluating cleanly with the service simply absent.

## What is already in SystemNix

- `modules/nixos/services/visionreviewd.nix` — lazy wrapper (package from the
  flake input, port from `lib/ports.nix`).
- `lib/ports.nix` — `visionreviewd-llama = 8390;`.
- `flake.nix` — input `vision-review-agent` (`github:LarsArtmann/vision-review-agent?ref=master`,
  following nixpkgs/flake-parts/systems/treefmt-nix) plus its lock entry.

## Activation steps

1. **Push this repo** so `master` on GitHub contains the module:
   `git push origin master`.
2. **Bump the SystemNix input**:
   `cd ~/projects/SystemNix && nix flake lock --update-input vision-review-agent`.
3. **Place the daemon config** at `/etc/visionreviewd/config.json` on the
   target host (generate a starting point with `visionreviewd discover ~/projects`).
   Point `dataDir` and `reviewsDir` under `/var/lib/visionreviewd`, and
   `baseUrl` at `http://127.0.0.1:8390/v1` when the llama unit is enabled:

   ```json
   {
     "model": "GitMylo/nsfwcaption-qwen3-vl-8b-v3-gguf:Q8_0",
     "baseUrl": "http://127.0.0.1:8390/v1",
     "dataDir": "/var/lib/visionreviewd/data",
     "reviewsDir": "/var/lib/visionreviewd/reviews",
     "projects": { "discordsync": ["/var/lib/discordsync/goldens/*.png"] }
   }
   ```

   Keep API keys out of store paths — manage the file via the host's secret
   tooling if it contains any.
4. **Enable on a host** (e.g. `systems/evo-x2.nix`):

   ```nix
   imports = [ nixosModules.visionreviewd ];  # or add to the host's module list
   services.vision-review-agent = {
     enable = true;
     configFile = "/etc/visionreviewd/config.json";
     llamaServer.enable = true;  # first start pulls ~9-10 GB of weights
   };
   ```

5. **Rebuild and verify**:

   ```bash
   nixos-rebuild switch --flake ~/projects/SystemNix#evo-x2
   systemctl status visionreviewd llama-vision-server
   journalctl -u visionreviewd -f
   ```

   The daemon's own health command doubles as a smoke test:
   `visionreviewd doctor -config /etc/visionreviewd/config.json`.

## Reviews for humans and agents

`reviewsDir` is plain markdown. Point it at a git-tracked checkout instead of
`/var/lib` if Crush should read the reviews from a repo — the writer does not
care what kind of directory it is; only the daemon user needs write access.
