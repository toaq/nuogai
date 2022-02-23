{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/master";
    flake-utils.url = "github:numtide/flake-utils/master";
    gomod2nix.url = "github:tweag/gomod2nix/master";
    nuigui-upstream.url = "github:uakci/nuigui/master";
    nuigui-upstream.flake = false;
    serial-predicate-engine-upstream.url =
      "github:acotis/serial-predicate-engine/master";
    serial-predicate-engine-upstream.flake = false;
  };

  outputs = { self, nixpkgs, gomod2nix, nuigui-upstream
    , serial-predicate-engine-upstream, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = (import nixpkgs {
          inherit system;
          overlays = [ gomod2nix.overlay ];
        }).pkgs;
      in with pkgs;
      let
        toaqScript = runCommand "toaq-script" { } ''
          mkdir -p $out/share/fonts
          cp ${./ToaqScript.ttf} $out/share/fonts/ToaqScript.ttf
        '';
        schemePkgs = lib.mapAttrs (name:
          { src, install, patches }:
          pkgs.stdenv.mkDerivation {
            inherit src name patches;
            buildInputs = [ guile ];
            installPhase = ''
              mkdir -p $out/bin
              cp -r ./* $out
              cp "${writers.writeBash "${name}-start" install}" $out/bin/${name}
            '';
          }) {
            nuigui = {
              src = nuigui-upstream;
              patches = [ ./patches/nui.patch ];
              install = ''
                cd $(dirname $0)/../
                ${guile}/bin/guile web.scm
              '';
            };
            serial-predicate-engine = {
              src = serial-predicate-engine-upstream;
              patches = [ ./patches/spe.patch ];
              install = ''
                cd $(dirname $0)/../web/
                ${guile}/bin/guile webservice.scm
              '';
            };
          };
        nuogai = buildGoApplication {
          vendorSha256 = null;
          runVend = true;
          name = "nuogai";
          src = ./.;
          modules = ./gomod2nix.toml;
          buildInputs = (builtins.attrValues schemePkgs) ++ [
            toaqScript
            (imagemagick.overrideAttrs
              (a: { buildInputs = a.buildInputs ++ [ pango ]; }))
          ];
        };
      in {
        defaultPackage = nuogai;
        packages = schemePkgs // { inherit toaqScript nuogai; };
        nixosModule = { config, pkgs, lib, ... }@args:
          import ./module.nix (args // { inherit self system; });
      }) // { inherit (gomod2nix) devShell; };
}
