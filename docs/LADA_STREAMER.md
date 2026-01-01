# Lada streaming helper (`lada-streamer`)

Stash supports a live streaming endpoint that runs an external helper binary and pipes its `stdout` directly to the HTTP response.

## Endpoint

- `GET /scenes/{sceneId}/stream-lada.webm?start=<seconds>`
- Alias: `GET /scenes/{sceneId}/stream.lada?start=<seconds>`
- Legacy alias: `GET /scenes/{sceneId}/stream-lada.mp4?start=<seconds>`

`start` is a floating-point number of seconds.

## Invocation

Stash will execute:

```text
lada-streamer[.exe]
```

If the environment variable `LADA_STREAMER_PATH` is set, Stash will use it as the executable path.

Example:

```text
lada-streamer.exe
```

## Seek mode

Stash uses Windows named pipes to keep a single `lada-streamer` process warm across requests. For each request, Stash creates a unique named pipe and sends:

```text
SEEK <seconds> <absolute-or-original-filepath> \\.\pipe\<pipeName>\n
```

`lada-streamer` is expected to connect to the provided pipe path and write **only** WebM bytes to it, then close the pipe when done.

## Required output behavior

To be readable by Stash (and browsers), `lada-streamer` must output **WebM** when writing to a pipe/non-seekable sink.

- **Output**:
  - Stash streams the **named pipe connection** to the HTTP response; stdout is ignored.
  - Current Stash implementation expects **WebM** (`Content-Type: video/webm`).
- **`stderr`**: write *only* errors.
  - Do **not** write progress logs to `stderr` on success.
  - Stash consumes `stderr` to avoid deadlocks; noisy progress output will pollute logs.
- **Exit codes**:
  - Exit `0` for success / normal completion.
  - Exit non-zero on failure.

## Notes

- Stash cancels the process via context when the client disconnects; your process should tolerate being terminated mid-stream.

