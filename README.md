# Zebra Printer Simulator

A virtual Zebra-style label printer. It listens on a TCP port like a real
networked thermal printer, accepts ZPL II jobs, renders each label to a PNG,
and exposes a small HTTP control plane (status JSON, fault injection, and a
browser dashboard) so applications can be developed and tested without
physical hardware.

The simulator targets the surface that most ZPL-emitting integrations actually
hit:

- Raw ZPL printing over a TCP socket (default port `19100`, the Zebra
  convention).
- The `~HS` Host Status response, returned as three Zebra-style framed lines.
- A small subset of SGD `! U1 getvar` queries (label counter, head DPI, print
  width).
- Fault-flag injection (paper out, paused, head up, ribbon out, under/over
  temperature) so client code can be exercised against unhappy printer states.

Rendering is delegated to [`github.com/ingridhq/zebrash`](https://github.com/ingridhq/zebrash);
the simulator wraps it with the network protocol, state machine, and dashboard.

## Quick start

Requires Go 1.25+.

```sh
make run               # go run ./cmd/simulator
```

In another shell:

```sh
nc localhost 19100 < demo.zpl
```

Then open <http://localhost:8081/> to see the rendered label appear in the
dashboard. PNGs are also written to `./output/`.

To build a static binary:

```sh
make build             # produces bin/simulator
```

### Docker

```sh
docker build -t printer-simulator .
docker run --rm -p 19100:19100 -p 8081:8081 \
  -v "$PWD/output:/output" \
  -e OUTPUT_DIR=/output \
  printer-simulator
```

## Configuration

All configuration is via environment variables. Defaults are shown.

| Variable           | Default     | Meaning                                                                         |
|--------------------|-------------|---------------------------------------------------------------------------------|
| `TCP_HOST`         | `""`        | Interface to bind the ZPL listener to (empty = all interfaces).                 |
| `TCP_PORT`         | `19100`     | Port the simulated printer listens on for ZPL jobs.                             |
| `HTTP_HOST`        | `""`        | Interface to bind the control API to (empty = all interfaces).                  |
| `HTTP_PORT`        | `8081`      | Port for the control API and dashboard.                                         |
| `OUTPUT_DIR`       | `./output`  | Directory where rendered label PNGs are written.                                |
| `LABEL_SIZE`       | `4x6`       | Fallback media size: `4x6`, `6x4`, or `2x4` (in inches).                        |
| `DPI`              | `203`       | Print resolution. One of `203` (8 dpmm), `300` (12 dpmm), `600` (24 dpmm).      |
| `BASIC_AUTH_USER`  | _(unset)_   | Username for HTTP basic auth. Must be set together with `BASIC_AUTH_PASS`.      |
| `BASIC_AUTH_PASS`  | _(unset)_   | Password for HTTP basic auth. When unset, the HTTP API is unauthenticated.      |
| `MAX_OUTPUT_FILES` | `0`         | Maximum PNGs to keep in `OUTPUT_DIR`. Oldest are deleted past this. `0` = no cap. |
| `WEBHOOK_URL`      | _(unset)_   | If set, POST a JSON event to this URL after every successful label render.      |

The default media size is only a fallback. If the incoming ZPL specifies
`^PW` (print width in dots) or `^LL` (label length in dots), the simulator
honors those instead — see `internal/size_detector.go`. The `DPI` setting
controls how those dot counts convert to millimetres.

## TCP protocol

The simulator accepts a long-lived TCP connection and reads requests until
the peer closes. Three kinds of input are recognised:

1. **ZPL labels.** Anything between `^XA` and `^XZ` is treated as one label
   format. Each format increments `formats_in_buffer` while it is rendering,
   then increments `label_count` on success. Multiple labels in one stream
   are supported — the buffer is scanned for the next `^XA`...`^XZ` pair.
2. **`~HS` (Host Status).** Returns the standard three-line Zebra status
   response, each line framed with `STX`/`ETX`, encoding the current
   fault-flag state and label counter. See `GenerateHSResponse` in
   `internal/status.go` for the exact field layout.
3. **`! U1 getvar "<name>"` and `! U1 setvar` (SGD).** A small set of getvars
   are answered (see below). All other SGD requests return `?` (Zebra's
   unknown-variable response). SGD responses can be globally disabled via
   the control API to test client behavior on older firmware.

Supported SGD getvars:

| Variable                       | Response                       |
|--------------------------------|--------------------------------|
| `odometer.total_label_count`   | Current rendered-label count.  |
| `head.resolution.in_dpi`       | `"203"`                        |
| `ezpl.print_width`             | `"832"` (832 dots = 4 in @203 dpi) |

Anything else returns `"?\r\n"`.

## Control API

The HTTP control plane is served on `HTTP_HOST:HTTP_PORT`.

| Method | Path       | Auth         | Purpose                                                                  |
|--------|------------|--------------|--------------------------------------------------------------------------|
| `GET`  | `/`        | basic auth   | HTML dashboard (embedded; see `internal/dashboard.html`).                |
| `GET`  | `/healthz` | always open  | Plain-text `ok` for liveness probes.                                     |
| `GET`  | `/status`  | basic auth   | JSON snapshot of printer state.                                          |
| `POST` | `/config`  | basic auth   | Toggle a fault flag or the SGD-enabled bit, or set the print speed.      |
| `POST` | `/reset`   | basic auth   | Clear counters, fault flags, and the output directory. Useful in CI.     |
| `GET`  | `/jobs`    | basic auth   | JSON list of rendered labels, newest first, with dimensions and sizes.   |
| `GET`  | `/metrics` | basic auth   | Prometheus text-format metrics (label totals, faults, render failures).  |
| `GET`  | `/images/` | basic auth   | Static file server over `OUTPUT_DIR` (used by the dashboard's `<img>`).  |

When `BASIC_AUTH_USER` and `BASIC_AUTH_PASS` are unset, every route is open.
When they are set, every route except `/healthz` requires HTTP basic auth.
`/healthz` is always open so container orchestrators can probe it without
credentials.

`/status` returns:

```json
{
  "paper_out": false,
  "paused": false,
  "head_up": false,
  "ribbon_out": false,
  "under_temp": false,
  "over_temp": false,
  "label_count": 12,
  "formats_in_buffer": 0,
  "render_failures": 0,
  "sgd_enabled": true,
  "print_speed": "fast"
}
```

`/config` accepts `{ "flag": "<name>", "enabled": <bool> }`. Valid flag names
are the six fault flags above plus `sgd` (toggles `sgd_enabled`). Example:

```sh
curl -X POST localhost:8081/config \
  -H 'content-type: application/json' \
  -d '{"flag":"paper_out","enabled":true}'
```

To change the print speed, send `{ "flag": "speed", "speed": "<name>" }` where
`<name>` is `fast`, `normal`, or `slow`. Slower speeds add an artificial render
delay per label. Example:

```sh
curl -X POST localhost:8081/config \
  -H 'content-type: application/json' \
  -d '{"flag":"speed","speed":"slow"}'
```

Once a fault is set, the next `~HS` response from the TCP port reflects it.
This is the primary way to test how a printing client handles error states
without unplugging a real printer.

`POST /reset` clears counters, faults, and every PNG in `OUTPUT_DIR`, leaving
the simulator in a clean default state. It's intended for test isolation
between scenarios in a CI run. Response body:

```json
{ "status": "reset", "deleted_files": 12 }
```

### Webhook payload

When `WEBHOOK_URL` is set, every successful render fires an asynchronous,
fire-and-forget POST with `Content-Type: application/json`:

```json
{
  "filename": "label_20260201_124530.123.png",
  "path": "/output/label_20260201_124530.123.png",
  "label_count": 17
}
```

Webhook failures are logged but never block or fail the rendering job.

## Dashboard

The browser dashboard at `/` auto-refreshes every 3 seconds. It shows:

- A status chip strip (Ready / Error, label counter, buffer depth, and any
  active fault flags).
- Toggle buttons for each fault flag (POST to `/config` under the hood).
- A print-speed selector (`Fast`, `Normal`, `Slow`). Slower speeds add an
  artificial per-label render delay to mimic a physical printer's feed rate.
  The choice is applied live via `/config` (`{"flag":"speed","speed":"slow"}`).
- A grid of every PNG in `OUTPUT_DIR`, newest first, with filename, timestamp,
  size, and pixel dimensions.

Click any thumbnail to open the full-resolution PNG.

## Layout

```
cmd/simulator/main.go        # process entry point: wiring + lifecycle
internal/config.go           # Config struct + env-var loader
internal/tcp_server.go       # TCP listener, buffered framing, command dispatch
internal/zpl_handler.go      # input classifier (ZPL / ~HS / SGD), SGD responder
internal/status.go           # PrinterState: counters, fault flags, ~HS encoder
internal/renderer.go         # zebrash wrapper: ZPL -> PNG -> OUTPUT_DIR
internal/size_detector.go    # parse ^PW / ^LL out of incoming ZPL
internal/retention.go        # OutputRetention: cap PNG count in OUTPUT_DIR
internal/webhook.go          # async POST to WEBHOOK_URL on label render
internal/control_api.go      # HTTP handlers, basic auth, /metrics, /reset
internal/dashboard.html      # embedded single-page dashboard
testdata/*.zpl               # sample ZPL formats for manual smoke testing
demo.zpl                     # minimal one-label demo
```

## Test data

`testdata/` contains synthetic ZPL captures useful for exercising the
renderer end-to-end:

- `label_0.zpl`, `label_1.zpl` — UPS-style 4x6 shipping labels (text + barcodes).
- `label_2.zpl`, `label_7.zpl` — FedEx-style 4x6 shipping labels with PDF417.
- `label_3.zpl`, `label_4.zpl`, `label_8.zpl` — USPS-style labels using a
  `^GFB` postage indicia.
- `label_5.zpl`, `label_6.zpl`, `label_9.zpl` — USPS labels using a `^GFA`
  postage indicia and a `^BX` Data Matrix.
- `label_2x4_0.zpl`, `label_2x4_1.zpl` — small 2x4 inventory/SKU labels.
- `label_gfa_0.zpl`, `label_gfa_1.zpl`, `label_gfa_2.zpl` — large `^GFA`-only
  bitmap labels, useful for benchmarking the renderer's image path.

All names, addresses, phone numbers, account numbers, and tracking numbers in
these files are fictional. Carrier names (UPS, FedEx, USPS) appear because
they describe the *format* of the label being exercised, not real shipments.

Pipe any of them at the simulator with `nc`:

```sh
nc localhost 19100 < testdata/label_0.zpl
```

## Testing

```sh
make test              # go test ./...
```

The unit tests cover the input classifier, the SGD responder, the
`PrinterState` fault-flag state machine, the config loader, basic auth on
the control API, `/metrics` rendering, and the retention policy.

## License

MIT — see `LICENSE`.
