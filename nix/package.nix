{ lib, buildGoModule, go_1_26, alsa-utils, stdenv }:

buildGoModule.override { go = go_1_26; } {
  pname = "spank";
  version = "0-unstable";

  src = lib.cleanSource ./..;

  # Run `nix build` to get the correct hash, then replace this placeholder
  vendorHash = "sha256-R63lSTQvwZ/zw2ccuwXk6rkuQ0t2Zs4mU3Trh+IxKf4=";

  ldflags = [ "-s" "-w" ];

  # Tests require Apple Silicon accelerometer hardware
  doCheck = false;

  # On Linux, wrap the binary so arecord is on PATH
  nativeBuildInputs = lib.optionals (!stdenv.isDarwin) [ alsa-utils ];

  meta = {
    description = "Yells 'ow!' when you slap the laptop";
    homepage = "https://github.com/taigrr/spank";
    license = lib.licenses.mit;
    platforms = [ "aarch64-darwin" "x86_64-linux" "aarch64-linux" ];
    mainProgram = "spank";
  };
}
