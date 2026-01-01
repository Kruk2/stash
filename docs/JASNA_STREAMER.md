# Jasna streaming helper (`jasna-cli`)

Stash supports live streaming endpoints that delegate transcoding to an external `jasna-cli` process which serves HLS over HTTP.

## Endpoints

| Route | Label | CLI args |
|---|---|---|
| `stream.jasna-180` | Jasna (180) | `--max-clip-size 180 --temporal-overlap 15` |
| `stream.jasna-90` | Jasna (90) | `--max-clip-size 90 --temporal-overlap 8` |
| `stream.jasna-180-unet` | Jasna (180 unet) | `--max-clip-size 180 --temporal-overlap 10 --secondary-restoration unet-4x` |
| `stream.jasna-180-rtx` | Jasna (180 RTX) | `--max-clip-size 180 --temporal-overlap 10 --secondary-restoration rtx-super-res` |

When accessed, Stash ensures `jasna-cli` is running with the correct variant args, tells it which file to open via REST API, waits for readiness, then redirects (HTTP 302) the client to `http://localhost:8765/stream.m3u8`.

Switching between variants restarts the process with the new CLI args.

## jasna-cli invocation

Stash executes:

```text
jasna-cli[.exe] --stream --no-browser --log-level info <variant-args...>
```

### Environment variables

- **`JASNA_CLI_PATH`** — executable path. Supports `exe + args` format, e.g. `python.exe -m jasna`. Falls back to searching `PATH` and the stash executable directory.
- **`JASNA_WORKING_DIR`** — working directory for the jasna-cli process.

## Lifecycle

- Stash starts `jasna-cli` on demand (first stream request).
- The process is kept alive and reused across requests for the same variant.
- Switching variants stops the current process and starts a new one.
- After **3 minutes** of inactivity, Stash stops the process.
- Stash calls `DELETE /open` on idle timeout and on shutdown.

## jasna-cli REST API contract (port 8765)

### `POST /open`

Tell jasna-cli to open a file for streaming. **Blocks until the HLS stream is ready.**

- **Request**: `{"path": "/absolute/path/to/video.mp4"}` or `{"path": "...", "start": 1483.7}` for seeking.
- **Response**: `200 {"status": "ready"}` or `500 {"error": "..."}`
- If already streaming another file, switches to the new one.
- If `start` > 0, seeks to that position (seconds) in the stream.

### `DELETE /open`

Stop current stream, release GPU/encoding resources. Idempotent.

- **Response**: `200 {"status": "stopped"}`

### `GET /stream.m3u8`

HLS manifest. Available after a successful `POST /open`.

### `GET /status`

Health/readiness check.

- **Response**: `200 {"streaming": true, "path": "..."}` or `200 {"streaming": false}`

### CORS

All responses must include `Access-Control-Allow-Origin: *` since the browser fetches HLS segments cross-origin (stash and jasna-cli run on different ports).

## Notes

- Stash cancels the process via context on shutdown; jasna-cli should tolerate being terminated mid-stream.
- No ffmpeg is required — jasna-cli handles all transcoding internally.
