# qBittorrent container guidance

## Ownership boundary

- `compose.yaml`, `auto_remove.sh`, `.env.example`, and `configure.sh` are the repository-owned desired state.
- `.env`, qBittorrent runtime configuration, WebUI credentials, logs, BT backup data, downloads, and cloud-drive contents remain local and must not be committed.
- Keep the completion hook mounted read-only as `/config/auto_remove.sh`; do not replace it with a host symlink whose target is outside the container mount namespace.

## Safety rules

- Consult current official qBittorrent and Docker Compose documentation before changing behavior.
- The hook must fail closed: never delete a qBittorrent task unless the current item moved successfully.
- Do not recursively delete `.torrent` files or operate on unrelated download paths.
- Do not use real downloads or cloud files as test fixtures.

## Validation

```bash
sh -n docker/qbittorrent/auto_remove.sh
bash -n docker/qbittorrent/configure.sh
./docker/qbittorrent/test.sh
docker compose -f docker/qbittorrent/compose.yaml config --quiet
./docker/qbittorrent/configure.sh --check
```
