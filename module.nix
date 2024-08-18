{ config, pkgs, lib, system, self, ... }:
let cfg = config.services.nuogai;
in with lib; {
  options.services.nuogai = {
    enable = mkEnableOption "Enables the nuogaı Discord Bot";
    hostInternal = mkOption { type = types.str; };
    hostExternal = mkOption { type = types.str; };
    zugaiHost = mkOption { type = types.str; };
    tokenPath = mkOption { type = types.path; };
  };
  config = {
    fonts.fonts = optionals cfg.enable [ self.packages.${system}.toaqScript ];
    systemd.services.nuogai = {
      inherit (cfg) enable;
      description = "Toaq Discord bot";
      wantedBy = [ "multi-user.target" ];
      wants = [ "network-online.target" ];
      environment = with cfg; {
        TOADUA_HOST_INTERNAL = hostInternal;
        TOADUA_HOST_EXTERNAL = hostExternal;
        ZUGAI_HOST = zugaiHost;
      };
      script = ''
        export NUOGAI_TOKEN=$(cat ${cfg.tokenPath})
        ${self.packages.${system}.nuogai}/bin/nuogai
      '';
    };
  };
}
