# Ghostty config guidance

## Scope

This directory is the source of truth for `~/.config/ghostty` on Yoshua's Mac. The live path is expected to be symlinked into this repo by `setup_mac.sh`.

Tracked files:

- `config`: Ghostty user config.
- `shaders/*.glsl`: vendored shader files from `0xhckr/ghostty-shaders`.
- `shaders/README.md`: shader source and switching notes.

Ignored files:

- `config-*.bak`: local timestamped backups created before edits.

## Workflow

Before changing `config`, verify the live file and repo file are the same target:

```bash
python3 - <<'PY'
import os
live = os.path.expanduser('~/.config/ghostty/config')
repo = '/Users/areslee/Documents/dev/config/yet-another-mac-config/.config/ghostty/config'
print(os.path.exists(live), os.path.exists(repo), os.path.samefile(live, repo))
PY
```

Consult Ghostty's local docs for options before editing:

```bash
/Applications/Ghostty.app/Contents/MacOS/ghostty +show-config --default --docs > /tmp/ghostty-default-docs.txt
```

Create an ignored backup before edits:

```bash
cp -p ~/.config/ghostty/config ~/.config/ghostty/config-$(date +%Y%m%d-%H%M%S).bak
```

Validate after edits:

```bash
/Applications/Ghostty.app/Contents/MacOS/ghostty +validate-config --config-file="$HOME/.config/ghostty/config"
git diff --check
git status --short .config/ghostty README.md
```

## Shaders

Default shader:

```ini
custom-shader = ~/.config/ghostty/shaders/cursor_blaze.glsl
custom-shader-animation = true
```

To switch shader, change only the `custom-shader` filename in `config`, then validate. Ghostty reports that `custom-shader` can be changed at runtime and affects open terminals, but a full app restart is still a good smoke test for visual changes.

When refreshing the shader collection, copy only `.glsl` files from upstream and update `shaders/README.md` with the source commit. Do not vendor upstream `.git/` or preview PNGs unless Yoshua explicitly wants them.
