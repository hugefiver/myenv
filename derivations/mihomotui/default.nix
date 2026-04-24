{ buildGoModule, lib }:
buildGoModule {
  pname = "mihomotui";
  version = "0.1.0";
  src = ./.;
  vendorHash = "sha256-xa2wygHA1Yd+wIgayQYeNnPj7UjA/As9YCSzo85viAs=";
  meta = {
    description = "TUI for mihomo external-controller API";
    mainProgram = "mihomotui";
    license = lib.licenses.mit;
  };
}
