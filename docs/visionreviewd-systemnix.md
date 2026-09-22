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

   The daemon's own health command doubles as a smoke test (5 checks:
   dataDir, reviewsDir, project globs, journal read-and-fold, model
   endpoint) and exits nonzero when anything fails:
   `visionreviewd doctor -config /etc/visionreviewd/config.json`.

## Activated on evo-x2 (2026-09-22) — what actually shipped

The first real enablement deviates from the generic steps above, in ways
other hosts may want to copy:

- **`llamaServer.enable = false`** — evo-x2 already serves the caption model
  through SystemNix's `llama-vlm-cap` socket-activated instance
  (`http://127.0.0.1:8128/v1`, CPU-only, idle-unloads after 2 h). An always-on
  llama unit would pin ~10 GB of RAM around the clock and re-enter the
  llama.cpp/ROCm wedge history on that host. The `visionreviewd-llama` port
  (8390) stays registered in SystemNix for hosts that do want the bundled
  unit.
- **Config file is generated, not hand-placed** — `environment.etc.
  "visionreviewd/config.json"` is built from a single site list (sourceURLs
  and per-site screenshot globs derive from the same attrset), so the config
  has no `~` entries the DynamicUser daemon could mis-expand: globs are
  absolute under `/home/<user>/.local/share/vision-review-agent/screenshots/`.
  It holds no secrets (loopback, keyless llama), which is what makes
  `environment.etc` acceptable.
- **Stable model id** — llama-server reports the model path as the
  `/v1/models` id unless `--alias` is passed; the daemon's `model` field and
  the `doctor` model check want a stable id, so the captioner instance runs
  with `--alias nsfwcaption-qwen3-vl-8b-v3`.
- **Fresh journal** — `dataDir`/`reviewsDir` point into the service
  StateDirectory (`/var/lib/visionreviewd/{data,reviews}`), deliberately NOT
  at the user-space `~/.local/share/vision-review-agent` journal the manual
  fleet passes still own.
- **Upstream flake builds again** — `buildGoModule` overrides
  `go = pkgs.go_1_27`; the go.mod `>= 1.27.1` floor had broken every nix
  build of this repo (packages and the test/lint apps alike).

## Backup the journal

The bbolt journal (`<dataDir>/events.db`) is the source of truth; the
markdown reviews are a projection `visionreviewd replay` can rebuild.
Back up the journal, not the markdown:

```bash
sudo systemctl stop visionreviewd            # one writer handle: stop first
sudo visionreviewd backup \
  -config /etc/visionreviewd/config.json \
  /var/backups/visionreviewd/events-$(date +%F).db
sudo systemctl start visionreviewd
```

`backup` takes a consistent snapshot via one bbolt read transaction and
fsyncs the output. `-wait 30s` extends the lock wait if the daemon was
not stopped. Verify a backup by restoring it into a scratch `dataDir`
and running `visionreviewd replay -config` against it — the projection
must rebuild byte-identically (INDEX timestamps included).

## Reviews for humans and agents

`reviewsDir` is plain markdown. Point it at a git-tracked checkout instead of
`/var/lib` if Crush should read the reviews from a repo — the writer does not
care what kind of directory it is; only the daemon user needs write access.
