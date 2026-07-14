{
  # yet-another-mac-config：多主机 Mac 声明式配置入口
  # 参考文档：
  #   - Determinate Nix: https://docs.determinate.systems/
  #   - nix-darwin:      https://github.com/nix-darwin/nix-darwin
  #   - Home Manager:    https://nix-community.github.io/home-manager/
  description = "Multi-host Mac configuration with Determinate Nix, nix-darwin, and Home Manager";

  inputs = {
    # 跟随 unstable 以便获取较新的 macOS 支持
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

    nix-darwin = {
      url = "github:nix-darwin/nix-darwin/master";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    home-manager = {
      url = "github:nix-community/home-manager/master";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    # oh-my-tmux 不是 flake；由 flake.lock 固定上游 revision，
    # Home Manager 只链接其只读主配置，自定义仍保留在仓库 tmux.conf.local。
    oh-my-tmux = {
      url = "github:gpakosz/.tmux";
      flake = false;
    };

    # 官方 Determinate module 负责协调 Determinate Nix 与 nix-darwin，
    # 同时提供 determinateNix.* 配置入口。不要让它 follows 仓库 nixpkgs，
    # 以便继续使用 Determinate 官方缓存的构建产物。
    determinate.url = "https://flakehub.com/f/DeterminateSystems/determinate/3";
  };

  outputs = inputs@{ nixpkgs, nix-darwin, home-manager, ... }:
    let
      hosts = import ./nix/hosts;

      mkDarwinConfiguration = hostname: host:
        let
          inherit (host) system username repoPath;
          extraModules = host.modules or [ ];
        in
        nix-darwin.lib.darwinSystem {
          inherit system;
          specialArgs = { inherit inputs hostname system username repoPath; };
          modules = [
            inputs.determinate.darwinModules.default
            {
              # 由 Determinate Nix 管理 Nix daemon 与 /etc/nix 配置，
              # 避免 nix-darwin 同时接管同一边界。
              determinateNix.enable = true;
            }
            ./nix/darwin
            home-manager.darwinModules.home-manager
            {
              home-manager.useGlobalPkgs = true;
              home-manager.useUserPackages = true;
              # 激活时如果目标文件已存在，先备份为 *.hm-backup，避免覆盖手写配置
              home-manager.backupFileExtension = "hm-backup";
              home-manager.extraSpecialArgs = {
                inherit inputs hostname system username repoPath;
              };
              home-manager.users.${username} = import ./nix/home;
            }
          ] ++ extraModules;
        };

      supportedSystems = nixpkgs.lib.unique (
        map (host: host.system) (builtins.attrValues hosts)
      );
    in
    {
      # 每台 Mac 都有显式 output；激活时必须选择真实主机名，避免 default 指向错误机器。
      darwinConfigurations = nixpkgs.lib.mapAttrs mkDarwinConfiguration hosts;

      # 首次安装还没有 darwin-rebuild 时，sync_mac.sh 从锁定的 nix-darwin input
      # 构建这个入口，避免临时运行未锁定的 master。
      packages = nixpkgs.lib.genAttrs supportedSystems (system: {
        darwin-rebuild = nix-darwin.packages.${system}.darwin-rebuild;
      });

      apps = nixpkgs.lib.genAttrs supportedSystems (system: {
        darwin-rebuild = {
          type = "app";
          program = "${nix-darwin.packages.${system}.darwin-rebuild}/bin/darwin-rebuild";
        };
      });
    };
}
