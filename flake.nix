{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/release-22.05";
    flake-utils.url = "github:numtide/flake-utils/master";
    gomod2nix.url = "github:tweag/gomod2nix/master";
    toaq-dictionary = { url = "github:toaq/dictionary/master"; flake = false; };
    zugai = { url = "github:toaq/zugai/main"; flake = false; };
  };

  outputs = { self, nixpkgs, gomod2nix, flake-utils, zugai, toaq-dictionary, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = (import nixpkgs {
          inherit system;
          overlays =
            [ gomod2nix.overlays.default (_: super: { go = super.go_1_17; }) ];
        }).pkgs;
      in with pkgs;
      let
        toaqScript = runCommand "toaq-script" { } ''
          mkdir -p $out/share/fonts
          cp ${./ToaqScript.ttf} $out/share/fonts/ToaqScript.ttf
        '';
        imagemagickWithPango = imagemagick.overrideAttrs
          (a: { buildInputs = a.buildInputs ++ [ pango ]; });
        expand-serial = runCommand "expand-serial" { } ''
          mkdir -p $out/bin
          tee >$out/bin/expand-serial <<EOF
          #!/bin/sh
          ${pkgs.python3}/bin/python ${zugai}/src/expand_serial.py \
            -d ${toaq-dictionary}/dictionary.json \
            -d ${zugai}/data/supplement.json \
            -- "\$@"
          EOF
          chmod +x $out/bin/expand-serial
        '';
        nuogai = buildGoApplication {
          vendorSha256 = null;
          runVend = true;
          name = "nuogai";
          src = ./.;
          modules = ./gomod2nix.toml;
          nativeBuildInputs = [ makeWrapper ];
          postFixup = ''
            wrapProgram $out/bin/nuogai --prefix PATH : ${
              lib.makeBinPath [ imagemagickWithPango expand-serial ]
            }
          '';
        };
      in {
        nixosModule = { config, pkgs, lib, ... }@args:
          import ./module.nix (args // { inherit self system; });
        devShells.${system} = gomod2nix.devShells.${system};
        defaultPackage = nuogai;
        packages = { inherit nuogai expand-serial toaqScript; };
      });
}
