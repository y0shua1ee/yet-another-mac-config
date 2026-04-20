{ ... }:
{
  # =============================================================================
  # Phase 3C：少量稳定 `system.defaults.*` 试点（保守首版）
  # =============================================================================
  #
  # 设计要点：
  #   - 只接管「长期几乎不改、与本机当前值一致、回滚成本极低」的默认项。
  #   - 每一项写入的值都与当前机器上 `defaults read` 结果保持一致，
  #     因此首次 switch 预期不会产生任何用户可感知的行为变化。
  #   - 不声明任何当前处于 unset 状态的 key（例如 `ApplePressAndHoldEnabled`、
  #     自动替换/自动引号/拼写相关），避免把未定义的行为固化为强意见。
  #   - 不触碰偏好型、设备相关、账号态相关的默认项（输入法、触控板、通知、
  #     窗口动画、Dock persistent items、登录项、能源管理等）。
  #
  # 刻意未纳入（保留给后续阶段单独评估）：
  #   - Dock: `autohide`、`persistent-apps`、`tilesize`、`orientation` 等
  #     —— 均属偏好漂移区间，不适合第一版锁死。
  #   - Finder: `_FXShowPosixPathInTitle`、`ShowHardDrivesOnDesktop` 等
  #     —— 不在本机常用集合内。
  #   - NSGlobalDomain: `ApplePressAndHoldEnabled`、`NSAutomaticSpellingCorrectionEnabled`、
  #     `NSAutomaticQuoteSubstitutionEnabled`、`NSAutomaticDashSubstitutionEnabled` 等
  #     —— 当前均为 unset；由 Nix 主动置位会改变系统默认行为，风险高于收益。
  #   - 触控板 / trackpad、窗口动画、通知中心、loginwindow、软件更新策略等类别。
  #
  # 参考：https://github.com/nix-darwin/nix-darwin/blob/master/modules/system/defaults
  system.defaults = {
    # Finder：与当前机器值一致，且用户长期保持此设置
    finder = {
      # 显示所有文件扩展名
      AppleShowAllExtensions = true;
      # 显示路径栏
      ShowPathbar = true;
      # 显示状态栏
      ShowStatusBar = true;
      # 默认视图：列表（Nlsv = list view）；其它取值：icnv/clmv/Flwv
      FXPreferredViewStyle = "Nlsv";
    };

    # Dock：仅接管一项极稳定的空间排序偏好
    dock = {
      # 关闭「按最近使用自动重排 Mission Control 空间」——与 AeroSpace 体验相关，长期为 false
      mru-spaces = false;
    };

    # 全局键盘重复速率：与当前机器值一致，长期稳定
    # 说明：KeyRepeat / InitialKeyRepeat 在 nix-darwin 中以「tick」为单位，与 macOS
    # 设置面板滑块不是同一量纲；这里直接沿用本机实际值，避免引入新的主观偏好。
    NSGlobalDomain = {
      KeyRepeat = 2;
      InitialKeyRepeat = 30;
    };
  };
}
