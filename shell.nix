with import <nixpkgs> {};
stdenv.mkDerivation rec {
  name = "env";
  env = buildEnv { name = name; paths = buildInputs; };
  buildInputs = [
    act
    dpkg
    elmPackages.elm
    elmPackages.elm-analyse
    elmPackages.elm-format
    elmPackages.elm-json
    elmPackages.elm-language-server
    elmPackages.elm-test
    go
    golint
    nodejs-16_x
    nodePackages.yarn
    rpm
    swaks
  ];
}
