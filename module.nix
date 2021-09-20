{ config, pkgs, lib, self, ... }:
let cfg = config.services.nuogai; in with lib; {
  options.services.nuogai = {
    enable = mkEnableOption "Enables the nuogaı Discord Bot";
    guilePackage = mkOption { default = pkgs.guile; type = types.package; };
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
        serviceConfig.WorkingDirectory = "${self.packages.nuigui.${system}}";
        serviceConfig.ExecStart = "${cfg.guilePackage} ./web.scm";
        environment.PORT = cfg.nuiPort;
      };
      serial-predicate-engine = {
        serviceConfig.WorkingDirectory = "${self.packages.serial-predicate-engine.${system}}";
        serviceConfig.ExecStart = "${cfg.guilePackage} ./web/webservice.scm";
        environment.PORT = cfg.spePort;
      };
    };
  };
}
