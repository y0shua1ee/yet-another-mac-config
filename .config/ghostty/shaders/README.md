# Ghostty shaders

Shader collection vendored from [`0xhckr/ghostty-shaders`](https://github.com/0xhckr/ghostty-shaders).

- Source commit: `aa6121ba2ddd5251ac75b92729c758fe41256e55`
- Installed files: top-level `*.glsl` files only
- Preview images and upstream `.git/` metadata are intentionally omitted

The active shader is configured in `../config`:

```ini
custom-shader = ~/.config/ghostty/shaders/cursor_blaze.glsl
custom-shader-animation = true
```

To switch effects, replace `cursor_blaze.glsl` with another file in this directory and validate:

```bash
/Applications/Ghostty.app/Contents/MacOS/ghostty +validate-config --config-file="$HOME/.config/ghostty/config"
```
