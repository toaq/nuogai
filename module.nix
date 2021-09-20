{ config, pkgs, lib, system, self, ... }:
let cfg = config.services.nuogai;
in with lib; {
  options.services.nuogai = {
    enable = mkEnableOption "Enables the nuogaı Discord Bot";
    ports = listToAttrs
      (map (flip attrsets.nameValuePair (mkOption { type = types.port; })) [
        "nuigui"
        "serial-predicate-engine"
        "toadua"
      ]);
    tokenPath = mkOption { type = types.path; };
  };
  config = mkIf cfg.enable {
    fonts.fonts = [ self.packages.${system}.toaqScript ];
    systemd.services = lib.mapAttrs (k: v:
      {
        wants = [ "network-online.target" ];
      } // (v self.packages.${system}.${k})) {
        nuogai = pkg: {
          description = "Toaq Discord bot";
          wantedBy = [ "multi-user.target" ];
          wants = [ "nuigui.service" "serial-predicate-engine.service" ];
          environment = {
            NUI_PORT = toString cfg.ports.nuigui;
            SPE_PORT = toString cfg.ports.serial-predicate-engine;
            TOA_PORT = toString cfg.ports.toadua;
          };
          script = ''
            export TOKEN=$(cat ${cfg.tokenPath})
            ${pkg}/bin/nuogai
          '';
        };
        nuigui = pkg: {
          serviceConfig.WorkingDirectory = pkg;
          serviceConfig.ExecStart = "${pkg}/bin/nuigui";
          environment.PORT = toString cfg.ports.nuigui;
        };
        serial-predicate-engine = pkg: {
          serviceConfig.WorkingDirectory = pkg;
          serviceConfig.ExecStart = "${pkg}/bin/serial-predicate-engine";
          environment.PORT = toString cfg.ports.serial-predicate-engine;
        };
      };
  };
}
