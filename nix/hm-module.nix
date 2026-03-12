flake:

{ config, lib, pkgs, ... }:

let
  cfg = config.programs.spank;
  spankPkg = flake.packages.${pkgs.stdenv.hostPlatform.system}.default;
in
{
  options.programs.spank = {
    enable = lib.mkEnableOption "spank - yells when you slap the laptop";

    package = lib.mkOption {
      type = lib.types.package;
      default = spankPkg;
      defaultText = lib.literalExpression "inputs.spank.packages.\${system}.default";
      description = "The spank package to use.";
    };

    mode = lib.mkOption {
      type = lib.types.enum [ "pain" "sexy" "halo" ];
      default = "pain";
      description = "Audio mode to use.";
    };

    fast = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Enable faster detection tuning.";
    };

    minAmplitude = lib.mkOption {
      type = lib.types.nullOr lib.types.float;
      default = null;
      description = "Minimum amplitude threshold (0.0-1.0).";
    };

    cooldown = lib.mkOption {
      type = lib.types.nullOr lib.types.int;
      default = null;
      description = "Cooldown between responses in milliseconds.";
    };

    volumeScaling = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Scale playback volume by slap amplitude.";
    };

    customPath = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = null;
      description = "Path to custom MP3 audio directory.";
    };

    micMode = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Use microphone-based knock detection (Linux only).";
    };

    micDevice = lib.mkOption {
      type = lib.types.str;
      default = "hw:0,6";
      description = "ALSA capture device for mic mode (use 'arecord -l' to list devices).";
    };

    micThreshold = lib.mkOption {
      type = lib.types.nullOr lib.types.float;
      default = null;
      description = "RMS amplitude threshold for mic mode (0.0-1.0, lower = more sensitive).";
    };
  };

  config = lib.mkIf cfg.enable {
    home.packages = [ cfg.package ];
  };
}
