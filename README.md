# dropla

Self-hosted file transfer. Upload files, get a link, send it.

- **Shares** - Upload files, recipients download individual files or a zip of everything.
- **Upload Boxes**  Send someone a link and they upload into it, without being able to see what is inside.

## Run

```sh
docker run -d --name dropla \
  -p 3000:3000 \
  -v dropla-data:/data \
  -e PUBLIC_BASE_URL=https://drop.example.com \
  ghcr.io/bwees/dropla:latest
```

```yaml
services:
  dropla:
    image: ghcr.io/bwees/dropla:latest
    ports:
      - '3000:3000'
    volumes:
      - ./data:/data
    environment:
      - PUBLIC_BASE_URL=https://drop.example.com
    restart: unless-stopped
```

Open it in a browser and create the first account, which becomes the administrator.

Transfers are unbounded in size, so behind a reverse proxy raise the body-size and timeout limits. Caddy needs no changes; nginx and Cloudflare do.

Where the limit cannot be raised — Cloudflare caps request bodies at 100MB below Enterprise — set `PUBLIC_MAX_REQUEST_BYTES` instead and uploads are split to fit. Only requests arriving on `PUBLIC_BASE_URL` are split, so reaching the same instance directly still sends each file in one request.

## Configuration

| Variable | Default | |
| --- | --- | --- |
| `PORT` | `3000` | Port to listen on |
| `PUBLIC_BASE_URL` | — | Public origin, e.g. `https://drop.example.com`. Set it to hand out links that work from outside your network |
| `DATABASE_DIR` | `/data/db` | Where the database lives |
| `UPLOAD_DIR` | `/data/uploads` | Where uploaded files live |
| `DEFAULT_USER_QUOTA_BYTES` | `0` | Storage per account, `0` for unlimited. Admins can override it per user |
| `MAX_UPLOAD_BYTES` | `0` | Largest single file, `0` for unlimited |
| `PUBLIC_MAX_REQUEST_BYTES` | `0` | Largest request body accepted on `PUBLIC_BASE_URL`, `0` to send each file in one request. Set it a little under the proxy's own limit — e.g. `95000000` behind Cloudflare |
| `SHARE_GRACE_PERIOD` | `168h` | How long files survive after a share expires, so a share can still be extended |
| `ABANDONED_UPLOAD_AFTER` | `48h` | When an unfinished upload is discarded |
| `SWEEP_INTERVAL` | `1h` | How often cleanup runs |

Point `DATABASE_DIR` and `UPLOAD_DIR` at separate volumes if you want the files on different storage from the database.

## Forgotten password

If the only administrator is locked out, reset from the container. The password is read from stdin so it stays out of your shell history:

```sh
echo 'new-password' | docker exec -i dropla dropla reset-password alice
```

## Develop

```sh
mise install
mise run install:web
mise run dev            # API on :3000, web on :5173,  open :5173
```

Or in containers: `docker compose -f docker-compose.dev.yaml up`

| Task | |
| --- | --- |
| `mise run dev` | Both dev servers |
| `mise run openapi` | Regenerate the API spec and web client after changing a handler |
| `mise run test` | Go tests |
| `mise run lint` | Vet, prettier, eslint |
| `mise run check` | Type-check the frontend |
| `mise run build` | Production binary and web bundle |
