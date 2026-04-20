{
  # yet-another-mac-config：渐进式 Nix 迁移骨架（Phase 1）
  # 参考文档：
  #   - Determinate Nix: https://docs.determinate.systems/
  #   - nix-darwin:      https://github.com/nix-darwin/nix-darwin
  #   - Home Manager:    https://nix-community.github.io/home-manager/
  description = "yet-another-mac-config: Determinate Nix + nix-darwin + Home Manager (Phase 1 skeleton)";

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
  };

  outputs = { self, nixpkgs, nix-darwin, home-manager, ... }@inputs:
    let
      # Phase 1 仅面向当前这台 Apple Silicon Mac；后续可扩展到多主机
      system = "aarch64-darwin";
      username = "areslee";
      hostname = "AresdeMacBook-Air";
    in
    {
      # 用法：darwin-rebuild switch --flake .#AresdeMacBook-Air
      # 首次激活前强烈建议：darwin-rebuild build --flake .#AresdeMacBook-Air
      darwinConfigurations.${hostname} = nix-darwin.lib.darwinSystem {
        inherit system;
        specialArgs = { inherit inputs username hostname; };
        modules = [
          ./nix/darwin
          home-manager.darwinModules.home-manager
          {
            home-manager.useGlobalPkgs = true;
            home-manager.useUserPackages = true;
            # 激活时如果目标文件已存在，先备份为 *.hm-backup，避免覆盖手写配置
            home-manager.backupFileExtension = "hm-backup";
            home-manager.extraSpecialArgs = { inherit inputs username; };
            home-manager.users.${username} = import ./nix/home;
          }
        ];
      };

      # 方便 `nix flake check` / CI 引用的快捷别名
      darwinConfigurations.default = self.darwinConfigurations.${hostname};
    };
}
