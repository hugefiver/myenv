{ buildGoModule, lib }:
buildGoModule {
  pname = "mihomotui";
  version = "0.1.0";
  src = ./.;
  vendorHash = lib.fakeHash;
  meta = {
    description = "TUI for mihomo external-controller API";
    mainProgram = "mihomotui";
    license = lib.licenses.mit;
  };
}
