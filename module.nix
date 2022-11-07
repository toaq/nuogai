{ config, pkgs, lib, system, self, ... }:
let cfg = config.services.nuogai;
in with lib; {
  options.services.nuogai = {
    enable = mkEnableOption "Enables the nuogaı Discord Bot";
    toaduaHost = mkOption { type = types.str; };
    zugaiHost = mkOption { type = types.str; };
    tokenPath = mkOption { type = types.path; };
  };
  config = {
    fonts.fonts = optionals cfg.enable [ self.packages.${system}.toaqScript ];
    services.nuogai.zugaiHost = lib.mkDefault "https://zugai.toaq.me";
    systemd.services.nuogai = {
      inherit (cfg) enable;
      description = "Toaq Discord bot";
      wantedBy = [ "multi-user.target" ];
      wants = [ "network-online.target" ];
      environment = {
        TOADUA_HOST = cfg.toaduaHost;
        ZUGAI_HOST = cfg.zugaiHost;
      };
      script = ''
        export NUOGAI_TOKEN=$(cat ${cfg.tokenPath})
        ${self.packages.${system}.nuogai}/bin/nuogai
      '';
    };
  };
}
