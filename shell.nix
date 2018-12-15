with import <nixpkgs> {};
stdenv.mkDerivation rec {
  name = "env";
  env = buildEnv { name = name; paths = buildInputs; };
  buildInputs = [
    elmPackages.elm
    elmPackages.elm-format
    go
    golint
    nodejs
    swaks
  ];
}
