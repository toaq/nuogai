{ config, pkgs, lib, self, ... }:
let cfg = config.services.nuogai; in with lib; {
  options.services.nuogai = {
    enable = mkEnableOption "Enables the nuogaı Discord Bot";
    guilePackage = mkOption { default = pkgs.guile; type = types.package; };
    nuiPort = mkOption { type = types.port; };
    spePort = mkOption { type = types.port; };
    tokenPath = mkOption { type = types.path; };
  };
  config = mkIf cfg.enable {
    fonts.fonts = [ self.packages.toaqScript.${system} ];
    systemd.services = lib.mapAttrs (k: v: {
      wants = [ "network-online.target" ];
    } // v) {
      nuogai = {
        description = "Toaq Discord bot";
        wantedBy = [ "multi-user.target" ];
        wants = [ "nuigui.service" "serial-predicate-engine.service" ];
        environment = {
          NUI_PORT = toString cfg.nuiPort;
          SPE_PORT = toString cfg.spePort;
        };
        script = ''
          export TOKEN=$(cat ${cfg.tokenPath})
          ${inputs.nuogai.packages.${system}.nuogai}/bin/nuogai
        '';
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
