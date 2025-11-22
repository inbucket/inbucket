{
  pkgs ? import <nixpkgs> { },
}:
let
  scripts = {
    # Quick test script.
    qt = pkgs.writeScriptBin "qt" ''
      # Builds & starts inbucket, then sends it some test mail.

      make build test inbucket || exit
      (sleep 3; etc/swaks-tests/run-tests.sh >/dev/null) &
      env INBUCKET_LOGLEVEL=debug ./inbucket
    '';
  };
in
pkgs.mkShell {
  buildInputs = with pkgs; [
    act
    dpkg
    delve
    elmPackages.elm
    elmPackages.elm-analyse
    elmPackages.elm-format
    elmPackages.elm-json
    elmPackages.elm-language-server
    elmPackages.elm-test
    go_1_25
    golangci-lint
    golint
    gopls
    nodejs_20
    nodePackages.node-gyp
    nodePackages.yarn
    rpm
    swaks

    scripts.qt
  ];

  # Prevents launch errors with delve debugger.
  hardeningDisable = [ "fortify" ];
}
