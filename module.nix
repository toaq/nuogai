{ config, lib, self, ... }:
let cfg = config.services.nuogai; in with lib; {
  options.services.nuogai = {
    enable = mkEnableOption "Enables the nuogaı Discord Bot";
    nuiPort = mkOption { type = types.port; };
    spePort = mkOption { type = types.port; };
  };
  config = mkIf cfg.enable {
    fonts.fonts = [ self.packages.toaqScript.${system} ];
    systemd.services = lib.mapAttrs (k: v: v // {
      wantedBy = [ "multi-user.target" ];
      wants = v.wants or [] ++ [ "network-online.target" ];
    }) {
      nuogai = {
        wants = [ "nuigui.service" "serial-predicate-engine.service" ];
        serviceConfig.ExecStart = "${self.packages.nuogai.${system}}/bin/nuogai";
        environment = {
          NUI_PORT = cfg.nuiPort;
          SPE_PORT = cfg.spePort;
        };
      };
      nuigui = {
        serviceConfig.ExecStart = "${self.packages.nuigui.${system}}/bin/nuigui";
        environment.PORT = cfg.nuiPort;
      };
      serial-predicate-engine = {
        serviceConfig.ExecStart = "${self.packages.serial-predicate-engine.${system}}/bin/serial-predicate-engine";
        environment.PORT = cfg.spePort;
      };
    };
  };
}
