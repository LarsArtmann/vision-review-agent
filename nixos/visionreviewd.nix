# NixOS module: visionreviewd, the event-sourced UI review daemon.
#
# Runs `visionreviewd run` as a systemd service and optionally provides the
# local llama-server vision model endpoint it talks to. The daemon reads a
# plain JSON config file (see `visionreviewd discover`) whose dataDir and
# reviewsDir should point under /var/lib/visionreviewd so the service's
# StateDirectory owns them.
self:
{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.services.vision-review-agent;
in
{
  options.services.vision-review-agent = {
    enable = lib.mkEnableOption "visionreviewd, the event-sourced UI review daemon";

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.visionreviewd;
      defaultText = "the flake's visionreviewd package";
      description = "The visionreviewd package to run.";
    };

    configFile = lib.mkOption {
      type = lib.types.path;
      example = "/etc/visionreviewd/config.json";
      description = ''
        Path to the daemon config JSON (generate a starting point with
        `visionreviewd discover`). The file must be readable by the service
        user. Point dataDir and reviewsDir under /var/lib/visionreviewd so
        the service's StateDirectory owns them.

        Beware: a path literal from the Nix store (like ./config.json) is
        world-readable in the store — put configs containing API keys into
        /etc or manage them with your secret tool instead.
      '';
    };

    llamaServer = {
      # Disabled by default: the first start pulls ~9-10 GB of model weights.
      enable = lib.mkEnableOption "the local llama-server vision model endpoint";

      package = lib.mkOption {
        type = lib.types.package;
        default = pkgs.llama-cpp;
        defaultText = lib.literalExpression "pkgs.llama-cpp";
        description = "The llama.cpp package providing llama-server.";
      };

      model = lib.mkOption {
        type = lib.types.str;
        default = "GitMylo/nsfwcaption-qwen3-vl-8b-v3-gguf:Q8_0";
        description = ''
          Hugging Face repository (with optional quantization suffix)
          llama-server downloads and serves. The daemon config's baseUrl
          should point at http://127.0.0.1:PORT/v1.
        '';
      };

      port = lib.mkOption {
        type = lib.types.port;
        default = 8390;
        description = "TCP port llama-server listens on (loopback only).";
      };
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.visionreviewd = {
      description = "Event-sourced UI review daemon";
      documentation = [ "https://github.com/LarsArtmann/vision-review-agent" ];
      wantedBy = [ "multi-user.target" ];
      after = lib.optionals cfg.llamaServer.enable [ "llama-vision-server.service" ];
      wants = lib.optionals cfg.llamaServer.enable [ "llama-vision-server.service" ];

      serviceConfig = {
        ExecStart = "${lib.getExe cfg.package} run -config ${toString cfg.configFile}";
        WorkingDirectory = "/var/lib/visionreviewd";
        DynamicUser = true;
        StateDirectory = "visionreviewd";
        Restart = "on-failure";
        RestartSec = "30s";

        NoNewPrivileges = true;
        ProtectSystem = "strict";
        # "read-only" (not "true") so a config file under /home stays readable.
        ProtectHome = "read-only";
        PrivateTmp = true;
        PrivateDevices = true;
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectControlGroups = true;
        ProtectClock = true;
        RestrictAddressFamilies = [
          "AF_INET"
          "AF_INET6"
          "AF_UNIX"
        ];
        RestrictNamespaces = "true";
        RestrictRealtime = true;
        LockPersonality = true;
        CapabilityBoundingSet = "";
        SystemCallArchitectures = "native";
        SystemCallFilter = [ "@system-service" ];
        # The model does the heavy lifting; the daemon itself only scans,
        # archives, and writes markdown.
        MemoryMax = "1G";
      };
    };

    systemd.services.llama-vision-server = lib.mkIf cfg.llamaServer.enable {
      description = "Local llama-server vision model endpoint for visionreviewd";
      wantedBy = [ "multi-user.target" ];

      serviceConfig = {
        # -hf pulls model weights from Hugging Face on first start.
        ExecStart =
          "${lib.getExe cfg.llamaServer.package} server "
          + "--host 127.0.0.1 --port ${toString cfg.llamaServer.port} -hf ${cfg.llamaServer.model}";
        # Readiness gate: llama-server's /health answers 503 until the
        # model is fully loaded, so this probe keeps the unit "activating"
        # until the endpoint actually serves. visionreviewd (after= and
        # wants= this unit) therefore cannot race the ~10 GB first-start
        # model download/load. Retries cover up to ~1h of pulling.
        ExecStartPost = [
          (
            "${lib.getExe pkgs.curl} --silent --fail --retry 720 --retry-delay 5"
            + " --retry-connrefused --retry-all-errors"
            + " http://127.0.0.1:${toString cfg.llamaServer.port}/health"
          )
        ];
        # The first start can spend a long time downloading weights; the
        # 90s default would kill it mid-download.
        TimeoutStartSec = "infinity";
        Environment = [ "HF_HOME=/var/lib/llama-vision-server/huggingface" ];
        WorkingDirectory = "/var/lib/llama-vision-server";
        DynamicUser = true;
        StateDirectory = "llama-vision-server";
        Restart = "on-failure";
        RestartSec = "10s";

        NoNewPrivileges = true;
        ProtectSystem = "strict";
        ProtectHome = "true";
        PrivateTmp = true;
        RestrictAddressFamilies = [
          "AF_INET"
          "AF_INET6"
          "AF_UNIX"
        ];
        RestrictNamespaces = "true";
        RestrictRealtime = true;
        LockPersonality = true;
        # Headroom for the ~9-10 GB of weights plus KV cache; cap generously
        # so a runaway inference cannot consume the host.
        MemoryMax = "16G";
      };
    };
  };
}
