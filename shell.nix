with import <nixpkgs> {};
stdenv.mkDerivation rec {
  name = "env";
  env = buildEnv { name = name; paths = buildInputs; };
  buildInputs = [
    dpkg
    elmPackages.elm
    elmPackages.elm-format
    go
    golint
    nodejs-10_x
    rpm
    swaks
  ];
}
