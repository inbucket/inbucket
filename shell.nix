with import <nixpkgs> {};
stdenv.mkDerivation rec {
  name = "env";
  env = buildEnv { name = name; paths = buildInputs; };
  buildInputs = [
    dpkg
    elmPackages.elm
    elmPackages.elm-analyse
    elmPackages.elm-format
    elmPackages.elm-language-server
    elmPackages.elm-test
    go
    golint
    nodejs-10_x
    rpm
    swaks
  ];
}
