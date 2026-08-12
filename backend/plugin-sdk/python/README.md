# Python Automation Plugin SDK

An SDK for writing `automation` plugins in Python that are loadable by the Go
host over the **Hashicorp go-plugin transport** (gRPC + magic-cookie handshake),
matching `backend/pkg/pluginsdk` and `backend/plugin/v1/*.proto`.

## What changed (P2-83)

The old SDK was a scaffold whose `run_local_stub` printed the manifest and
**exited**, so it could never be loaded by the Go host. It is now a real
transport:

- `transport.py` implements the go-plugin wire contract:
  - magic-cookie handshake validation (`XIAOHEI_PLUGIN=xiaoheiplay`);
  - go-plugin `AutoMTLS` mutual authentication using `PLUGIN_CLIENT_CERT` and
    the server certificate returned in the handshake;
  - private Unix socket placement under the host-provided
    `PLUGIN_UNIX_SOCKET_DIR` (the host owns and removes that `0700` directory);
  - a gRPC server on a Unix socket (or TCP with `PLUGIN_MIN_PORT` /
    `PLUGIN_MAX_PORT`);
  - the stdout handshake line `1|1|unix|<addr>|grpc|` (stdout carries nothing
    else; diagnostics go to stderr);
  - graceful shutdown on SIGTERM/SIGINT and socket cleanup.
- `grpc_service.py` bridges the `CoreService` / `AutomationService` Python
  interfaces to the `plugin.v1` gRPC services; plugin exceptions become
  `grpc.StatusCode.INTERNAL` (raw messages are never sent on the wire).
- `server.serve(plugin)` is the production entrypoint (called by the Go host or
  an example `main()`); `python -m xiaohei_plugin_sdk` runs a plugin directly.
- Generated proto stubs are committed under `src/plugin/v1/` and regenerated
  with `python3 generate_protos.py`.

## Install

```bash
cd backend/plugin-sdk/python
pip install -e .
# optional: regenerate proto stubs after a proto change
pip install -e '.[dev]' && PYTHONPATH=src python3 generate_protos.py
```

## Run the example (real server, dev mode — no Go host)

```bash
cd backend/plugin-sdk/python
PYTHONPATH=src python3 -m xiaohei_plugin_sdk \
    --plugin-impl examples.echo_plugin:build_plugin --dev
```

This binds a Unix socket, prints the handshake line to stdout, and serves
`CoreService` + `AutomationService` until SIGTERM/SIGINT — it does not exit.

## Tests

```bash
cd backend/plugin-sdk/python
PYTHONPATH=src python3 -m unittest discover -s tests -p 'test_*.py' -v
```

The CI gate installs the project dependencies and additionally runs a Go test
that launches the Python example through a real `plugin.NewClient` configured
with `AutoMTLS` and `UnixSocketConfig`, then calls `Core.GetManifest`.

## Security notes

- Production launches require go-plugin AutoMTLS. Missing or malformed
  `PLUGIN_CLIENT_CERT` data fails before the SDK prints a handshake address;
  plaintext transport is available only through the explicit local `--dev`
  mode. Sensitive request fields are encrypted in transit but remain plaintext
  in plugin process memory.
- Python's standard-library OpenSSL endpoint terminates and verifies the
  host-pinned mTLS connection, then forwards the authenticated HTTP/2 stream to
  a separate private Unix socket used by grpcio. This is required because
  grpcio's bundled TLS verifier does not accept go-plugin's one-time
  self-signed client certificate consistently across supported platforms.
- Never log those values. Do not include passwords in exceptions or in the
  `error_message` of `OperationResult`. The transport strips raw exception
  text from gRPC errors, but plugin code must still avoid leaking secrets.
- Only the handshake line may be written to stdout; all other output must go
  to stderr, otherwise the Go host mis-parses the handshake.
