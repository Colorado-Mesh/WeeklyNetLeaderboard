# MeshMonday

Persistent web service for MeshCore Monday check-ins in the Cascadia Mesh region.

## Features
- MQTT ingestion for `meshcore/+/+/packets` (all IATA regions) with topic metadata extraction per packet
- MeshCore packet decoding (header/path/payload extraction)
- Monday check-in board with DiceBear avatars and animated tiles
- Leaderboard with:
  - Most check-ins (tracked from configurable date)
  - Longest Monday streak (with streak start date)
- SQLite storage with WAL mode and dedupe by packet hash

## Local macOS development (non-Docker)
1. `make setup`
2. Start an MQTT broker locally (`localhost:1883`) or point `.env.local` to your broker.
3. Run app once: `make run`
4. Run with hot reload: `make dev` (auto-installs `air` if needed)
5. Seed sample data (optional): `make dev-seed`
6. Replay fixtures (optional): `make replay-checkins`
7. Rebuild checkin packet links from stored raw packets (optional): `make restore-checkin-packets`

Open `http://127.0.0.1:8080`.

### Channel decryption keys
- Public channel key is built in: `8b3387e9c5cdea6ac9e5edbaa115cd72`.
- Deterministic hashtag channels: set `HASHTAG_CHANNELS` (comma-separated names, with or without `#`).
- Private channels: set `PRIVATE_CHANNEL_KEYS` as comma-separated 16-byte hex keys.

### UI settings
- DiceBear avatar style: set `DICEBEAR_STYLE` to any DiceBear style slug (e.g. `rings`, `adventurer`, `pixel-art`, `bottts`).
- Auto-refresh polling interval: set `UI_POLL_SECONDS` (default `15`, set `0` to disable).

### IATA filtering
- Ingest subscribes to all IATAs via `meshcore/+/+/packets`.
- Display/query filtering is controlled by `IATA_FILTERS`:
  - `ALL` for all regions
  - comma-separated list such as `SEA,PDX,YVR`

## Key paths
- Server entrypoint: `cmd/server/main.go`
- MQTT replay utility: `cmd/replay/main.go`
- Seed utility: `cmd/devseed/main.go`
- SQLite + migrations: `internal/storage/sqlite.go`
- Decode logic: `internal/meshcore/decode.go`
- Monday page: `web/templates/index.html`
- Leaderboard page: `web/templates/leaderboard.html`

## Tests
- `make test`
- `make test-integration`

## Docker for production
1. Create `.env` with production values.
2. Run `docker compose up -d --build`

## Backups
Backup helper: `scripts/backup_sqlite.sh`.

## Maintenance
- Rebuild `checkin_packets` from `raw_packets` with current parsing/decryption rules:
  - `make restore-checkin-packets`
- This is useful after schema cleanup or if packet-link rows were lost.

Example nightly cron:
`0 2 * * * cd /opt/meshmonday && ./scripts/backup_sqlite.sh ./data/meshmonday_prod.db ./backups`
