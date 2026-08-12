from __future__ import annotations

"""Hashicorp go-plugin transport bridge for the Python SDK.

Implements the wire contract the Go host (`backend/pkg/pluginsdk`) expects:

1. **Magic cookie handshake**: the host launches the plugin process with
   ``os.environ[MagicCookieKey]`` set; the plugin must validate it before doing
   anything else. A mismatch (the binary was executed directly by a human)
   prints a diagnostic to stderr and exits non-zero — never proceeds.

2. **gRPC server on a Unix socket** (or TCP when ports are configured): the
   plugin serves the ``plugin.v1.CoreService`` and ``plugin.v1.AutomationService``
   RPCs generated from ``backend/plugin/v1/*.proto``.

3. **Handshake line on stdout**: ``1|1|unix|<addr>|grpc|<cert>``. ``cert`` is
   populated for AutoMTLS. This is the ONLY thing written to stdout; all
   diagnostics go to stderr, otherwise the host mis-parses the handshake.

4. **Lifecycle**: serves until SIGTERM/SIGINT, then drains in-flight RPCs
   (graceful stop) and removes the socket file.

Error mapping: plugin methods that raise are turned into gRPC status errors
(``grpc.StatusCode.INTERNAL``) with the message stripped of any secret-bearing
exception text; the RPC-level error_code/error_message fields in
``OperationResult`` are left to the plugin implementation.
"""

import base64
import datetime
import itertools
import os
import select
import signal
import socket
import ssl
import sys
import tempfile
import threading
import time
from dataclasses import dataclass
from typing import Optional

# Aligned with backend/pkg/pluginsdk/handshake.go
MAGIC_COOKIE_KEY = "XIAOHEI_PLUGIN"
MAGIC_COOKIE_VALUE = "xiaoheiplay"
PROTOCOL_VERSION = 1
CORE_PROTOCOL_VERSION = 1

# Env vars set by go-plugin's client when spawning the plugin.
ENV_PROTOCOL_VERSIONS = "PLUGIN_PROTOCOL_VERSIONS"
ENV_MIN_PORT = "PLUGIN_MIN_PORT"
ENV_MAX_PORT = "PLUGIN_MAX_PORT"
ENV_CLIENT_CERT = "PLUGIN_CLIENT_CERT"
ENV_UNIX_SOCKET_DIR = "PLUGIN_UNIX_SOCKET_DIR"


class HandshakeError(Exception):
    """Raised when the go-plugin handshake environment is invalid."""


@dataclass(frozen=True)
class PluginEnv:
    """Parsed go-plugin environment (as set by the Go host)."""

    magic_cookie_key: str
    magic_cookie_value: str
    protocol_versions: tuple[int, ...]
    min_port: Optional[int]
    max_port: Optional[int]
    client_cert_pem: str
    unix_socket_dir: Optional[str]


def parse_env(environ: Optional[dict] = None) -> PluginEnv:
    """Parse the go-plugin environment.

    go-plugin communicates the cookie by setting the configured cookie key
    itself (``XIAOHEI_PLUGIN=xiaoheiplay``); it does not send separate key and
    value variables. A missing cookie remains empty so
    :func:`validate_handshake` can reject direct production launches.
    """
    env = dict(os.environ if environ is None else environ)

    value = env.get(MAGIC_COOKIE_KEY, "")
    key = MAGIC_COOKIE_KEY if value else ""

    versions = env.get(ENV_PROTOCOL_VERSIONS, "")
    parsed_versions: tuple[int, ...] = tuple()
    if versions.strip():
        try:
            parsed_versions = tuple(int(v.strip()) for v in versions.split(",") if v.strip())
        except ValueError as exc:  # pragma: no cover - defensive
            raise HandshakeError(f"malformed {ENV_PROTOCOL_VERSIONS}: {versions!r}") from exc

    def _port(name: str) -> Optional[int]:
        raw = env.get(name, "")
        if not raw.strip():
            return None
        try:
            port = int(raw)
        except ValueError as exc:
            raise HandshakeError(f"malformed {name}: {raw!r}") from exc
        # go-plugin emits zero for its unset MinPort/MaxPort defaults. It means
        # "use a Unix socket", not an invalid configured TCP port.
        if port == 0:
            return None
        if port < 0 or port > 65535:
            raise HandshakeError(f"{name} out of range: {port}")
        return port

    socket_dir = env.get(ENV_UNIX_SOCKET_DIR, "").strip() or None

    return PluginEnv(
        magic_cookie_key=key,
        magic_cookie_value=value,
        protocol_versions=parsed_versions,
        min_port=_port(ENV_MIN_PORT),
        max_port=_port(ENV_MAX_PORT),
        client_cert_pem=env.get(ENV_CLIENT_CERT, ""),
        unix_socket_dir=socket_dir,
    )


def validate_handshake(environ: Optional[dict] = None, *, allow_dev: bool = False) -> PluginEnv:
    """Validate the go-plugin magic cookie.

    Raises :class:`HandshakeError` when the cookie is missing/mismatched and
    ``allow_dev`` is false. ``allow_dev`` is used by the local-dev entrypoint
    (``python -m xiaohei_plugin_sdk`` without a Go host), where no cookie is
    expected; production host launches always carry the cookie.
    """
    env = parse_env(environ)
    if env.magic_cookie_value == "" or env.magic_cookie_key == "":
        if allow_dev:
            return env
        raise HandshakeError("plugin launched without a magic cookie (not run by the host)")
    if env.magic_cookie_value != MAGIC_COOKIE_VALUE:
        raise HandshakeError(
            f"invalid magic cookie for {env.magic_cookie_key!r} "
            f"(expected {MAGIC_COOKIE_VALUE!r}, got {env.magic_cookie_value!r})"
        )
    return env


def build_handshake_line(
    addr: str,
    *,
    network: str = "unix",
    proto_type: str = "grpc",
    server_cert: str = "",
) -> str:
    """Build the go-plugin handshake line printed to stdout.

    Format (go-plugin client.go `strings.Split(line, "|")`):
    ``core_protocol|proto_version|network|addr|proto_type|cert``. Under
    AutoMTLS, ``cert`` is the unpadded standard-base64 form of the server's raw
    DER leaf certificate, exactly as go-plugin's Go server emits it.
    """
    return f"{CORE_PROTOCOL_VERSION}|{PROTOCOL_VERSION}|{network}|{addr}|{proto_type}|{server_cert}"


_socket_counter = itertools.count(1)


def _default_socket_path(env: Optional[PluginEnv] = None) -> str:
    # Unique per process AND per server instance so multiple in-process servers
    # (tests, embedded usage) never clobber each other's socket.
    socket_dir = env.unix_socket_dir if env and env.unix_socket_dir else tempfile.gettempdir()
    if env and env.unix_socket_dir:
        if not os.path.isabs(socket_dir):
            raise HandshakeError(f"{ENV_UNIX_SOCKET_DIR} must be an absolute path")
        if not os.path.isdir(socket_dir):
            raise HandshakeError(f"{ENV_UNIX_SOCKET_DIR} is not a directory")
    return os.path.join(
        socket_dir,
        f"xiaohei-plugin-{os.getpid()}-{next(_socket_counter)}.sock",
    )


def _auto_mtls_credentials(client_cert_pem: str):
    """Create one-time server credentials compatible with go-plugin AutoMTLS."""
    try:
        from cryptography import x509
        from cryptography.hazmat.primitives import hashes, serialization
        from cryptography.hazmat.primitives.asymmetric import ec
        from cryptography.x509.oid import ExtendedKeyUsageOID, NameOID
    except ImportError as exc:  # pragma: no cover - packaging installs dependencies
        raise HandshakeError("AutoMTLS dependencies are unavailable") from exc

    try:
        # Parse before constructing credentials so malformed host input fails
        # synchronously, before a handshake address is printed.
        x509.load_pem_x509_certificate(client_cert_pem.encode("ascii"))
    except (ValueError, UnicodeError) as exc:
        raise HandshakeError(f"malformed {ENV_CLIENT_CERT}") from exc

    now = datetime.datetime.now(datetime.timezone.utc)
    key = ec.generate_private_key(ec.SECP521R1())
    subject = issuer = x509.Name(
        [
            x509.NameAttribute(NameOID.COMMON_NAME, "localhost"),
            x509.NameAttribute(NameOID.ORGANIZATION_NAME, "HashiCorp"),
        ]
    )
    cert = (
        x509.CertificateBuilder()
        .subject_name(subject)
        .issuer_name(issuer)
        .public_key(key.public_key())
        .serial_number(x509.random_serial_number())
        .not_valid_before(now - datetime.timedelta(seconds=30))
        .not_valid_after(now + datetime.timedelta(days=3650))
        .add_extension(x509.SubjectAlternativeName([x509.DNSName("localhost")]), critical=False)
        .add_extension(x509.BasicConstraints(ca=True, path_length=None), critical=True)
        .add_extension(
            x509.ExtendedKeyUsage([ExtendedKeyUsageOID.CLIENT_AUTH, ExtendedKeyUsageOID.SERVER_AUTH]),
            critical=False,
        )
        .sign(key, hashes.SHA512())
    )
    cert_pem = cert.public_bytes(serialization.Encoding.PEM)
    key_pem = key.private_bytes(
        serialization.Encoding.PEM,
        serialization.PrivateFormat.TraditionalOpenSSL,
        serialization.NoEncryption(),
    )
    server_cert = base64.b64encode(cert.public_bytes(serialization.Encoding.DER)).decode("ascii").rstrip("=")
    # grpcio's BoringSSL server rejects go-plugin's self-signed P-521 client
    # certificate before application hooks run. Terminate mTLS with CPython's
    # OpenSSL (strict CERT_REQUIRED), then proxy the authenticated HTTP/2 stream
    # to a private local gRPC socket.
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    context.minimum_version = ssl.TLSVersion.TLSv1_2
    context.set_alpn_protocols(["h2"])
    context.verify_mode = ssl.CERT_REQUIRED
    context.load_verify_locations(cadata=client_cert_pem)
    with tempfile.TemporaryDirectory(prefix="xiaohei-plugin-cert-") as cert_dir:
        cert_path = os.path.join(cert_dir, "server.pem")
        key_path = os.path.join(cert_dir, "server.key")
        with open(cert_path, "wb") as cert_file:
            cert_file.write(cert_pem)
        with open(key_path, "wb") as key_file:
            key_file.write(key_pem)
        context.load_cert_chain(cert_path, key_path)
    return context, server_cert


class _TLSTerminatingProxy:
    """Authenticate the host, then bridge TLS bytes to local gRPC."""

    def __init__(self, *, network: str, addr: str, backend_path: str, context: ssl.SSLContext):
        family = socket.AF_UNIX if network == "unix" else socket.AF_INET
        self._listener = socket.socket(family, socket.SOCK_STREAM)
        if network == "unix":
            _unlink_quiet(addr)
            self._listener.bind(addr)
            self.addr = addr
        else:
            host, port = addr.rsplit(":", 1)
            self._listener.bind((host, int(port)))
            bound_host, bound_port = self._listener.getsockname()[:2]
            self.addr = f"{bound_host}:{bound_port}"
        self._listener.listen(16)
        self._context = context
        self._backend_path = backend_path
        self._closed = threading.Event()
        self._thread = threading.Thread(target=self._accept, name="xiaohei-plugin-mtls", daemon=True)

    def start(self) -> None:
        self._thread.start()

    def stop(self) -> None:
        self._closed.set()
        try:
            self._listener.close()
        except OSError:
            pass

    def _accept(self) -> None:
        while not self._closed.is_set():
            try:
                connection, _ = self._listener.accept()
            except OSError:
                return
            threading.Thread(target=self._bridge, args=(connection,), daemon=True).start()

    def _bridge(self, connection: socket.socket) -> None:
        try:
            peer = self._context.wrap_socket(connection, server_side=True)
            backend = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            backend.connect(self._backend_path)
            sockets = (peer, backend)
            while not self._closed.is_set():
                readable, _, _ = select.select(sockets, (), (), 0.5)
                for source in readable:
                    data = source.recv(64 * 1024)
                    if not data:
                        return
                    (backend if source is peer else peer).sendall(data)
        except (OSError, ssl.SSLError):
            pass
        finally:
            for item in (locals().get("peer"), locals().get("backend"), connection):
                if item is not None:
                    try:
                        item.close()
                    except OSError:
                        pass


def serve(
    plugin,
    *,
    network: Optional[str] = None,
    addr: Optional[str] = None,
    block: bool = True,
    environ: Optional[dict] = None,
    allow_dev: bool = False,
    plugins: Optional[dict] = None,
) -> object:
    """Start a go-plugin gRPC server and serve [plugin].

    Args:
        plugin: a ``CoreService``/``AutomationService`` implementation (the
            same object the Go host would wrap as the ``automation`` plugin).
        network: ``"unix"`` (default when ``PLUGIN_UNIX_SOCKET_DIR`` is set)
            or ``"tcp"``. Otherwise configured ``PLUGIN_MIN_PORT`` /
            ``PLUGIN_MAX_PORT`` select TCP.
        addr: explicit socket address. Defaults to a per-pid Unix socket in the
            temp dir.
        block: if true, serve forever (until SIGTERM/SIGINT). If false, returns
            immediately after binding (for tests / callers that manage their
            own event loop); the caller must call ``stop()``.
        environ: environment override for tests.
        allow_dev: permit running without a host-provided magic cookie.
        plugins: extra go-plugin key -> gRPC servicer mapping (e.g.
            ``{"core": core_servicer, "automation": auto_servicer}``). When
            None, the services are derived from [plugin].

    Returns:
        A ``_ServerHandle`` with ``addr``, ``stop()`` and ``wait()``.
    """
    import grpc  # optional dependency: only needed when actually serving

    from . import grpc_service

    env = validate_handshake(environ=environ, allow_dev=allow_dev)

    production_launch = bool(env.magic_cookie_value)
    if production_launch and not env.client_cert_pem:
        raise HandshakeError(f"production host launch requires {ENV_CLIENT_CERT}")
    tls_context = None
    server_cert = ""
    if env.client_cert_pem:
        tls_context, server_cert = _auto_mtls_credentials(env.client_cert_pem)

    if network is None:
        network = "unix" if env.unix_socket_dir else ("tcp" if (env.min_port is not None or env.max_port is not None) else "unix")
    if addr is None:
        if network == "unix":
            addr = _default_socket_path(env)
        else:
            addr = "127.0.0.1:0"

    server = grpc.server(grpc_executor_factory())
    servicers = dict(plugins or {})
    servicers.setdefault("core", grpc_service.CoreServicer(plugin))
    servicers.setdefault("automation", grpc_service.AutomationServicer(plugin))
    grpc_service.register_all(server, servicers)

    proxy = None
    internal_socket = None
    if tls_context:
        internal_socket = f"{_default_socket_path(env)}.grpc"
        _unlink_quiet(internal_socket)
        bound = server.add_insecure_port(f"unix:{internal_socket}")
        if bound == 0:
            raise RuntimeError("failed to bind internal gRPC socket")
        proxy = _TLSTerminatingProxy(network=network, addr=addr, backend_path=internal_socket, context=tls_context)
        resolved_addr = proxy.addr
    elif network == "unix":
        _unlink_quiet(addr)
        target = f"unix:{addr}"
        bound = server.add_insecure_port(target)
        if bound == 0:
            raise RuntimeError(f"failed to bind unix socket {addr!r}")
        resolved_addr = addr
    else:
        bound = server.add_insecure_port(addr)
        if bound == 0:
            raise RuntimeError(f"failed to bind {addr!r}")
        resolved_addr = f"127.0.0.1:{bound}"

    handle = _ServerHandle(
        server=server,
        addr=resolved_addr,
        network=network,
        socket_path=addr if network == "unix" else None,
        proxy=proxy,
        internal_socket=internal_socket,
    )

    # go-plugin calls this control RPC from Client.Kill. Register the standard
    # service even though its proto is internal to go-plugin: both request and
    # response are empty protobuf messages, so the checked-in v1 Empty type is
    # wire-compatible. Stop just after returning the response so the host sees
    # a successful close before the transport disappears.
    from plugin.v1 import types_pb2

    def shutdown(_request, _context):
        def stop_after_response() -> None:
            time.sleep(0.05)
            handle.stop()

        threading.Thread(target=stop_after_response, name="xiaohei-plugin-shutdown", daemon=True).start()
        return types_pb2.Empty()

    server.add_generic_rpc_handlers(
        (
            grpc.method_handlers_generic_handler(
                "plugin.GRPCController",
                {
                    "Shutdown": grpc.unary_unary_rpc_method_handler(
                        shutdown,
                        request_deserializer=types_pb2.Empty.FromString,
                        response_serializer=types_pb2.Empty.SerializeToString,
                    )
                },
            ),
        )
    )

    # The gRPC server must be accepting connections before the host reads the
    # address, otherwise a fast local Unix socket dial can race startup.
    server.start()
    handle._started = True
    if proxy:
        proxy.start()

    # Print the handshake line ONLY after the listener is bound and serving;
    # everything else goes to stderr.
    sys.stdout.write(build_handshake_line(resolved_addr, network=network, server_cert=server_cert) + "\n")
    sys.stdout.flush()
    _install_signal_handlers(handle)
    if block:
        handle.wait()
    return handle


def grpc_executor_factory():
    """Return a ThreadPoolExecutor sized for concurrent plugin RPCs."""
    import concurrent.futures

    return concurrent.futures.ThreadPoolExecutor(max_workers=16)


def _unlink_quiet(path: str) -> None:
    try:
        os.unlink(path)
    except FileNotFoundError:
        pass
    except OSError:
        pass


class _ServerHandle:
    def __init__(
        self,
        *,
        server,
        addr: str,
        network: str,
        socket_path: Optional[str],
        proxy: Optional[_TLSTerminatingProxy] = None,
        internal_socket: Optional[str] = None,
    ):
        self._server = server
        self.addr = addr
        self.network = network
        self._socket_path = socket_path
        self._proxy = proxy
        self._internal_socket = internal_socket
        self._started = False

    def wait(self) -> None:
        self._server.wait_for_termination()

    def stop(self, grace: float = 2.0) -> None:
        if self._proxy:
            self._proxy.stop()
        self._server.stop(grace)
        if self._socket_path:
            _unlink_quiet(self._socket_path)
        if self._internal_socket:
            _unlink_quiet(self._internal_socket)

    def start(self) -> None:
        if not self._started:
            self._server.start()
            self._started = True

    # --- context manager support for tests / embedded usage ---
    def __enter__(self):
        if not self._started:
            self.start()
        return self

    def __exit__(self, *exc) -> None:
        self.stop()


def _install_signal_handlers(handle: _ServerHandle) -> None:
    def _shutdown(_signum, _frame):
        handle.stop()

    for sig in (signal.SIGTERM, signal.SIGINT):
        try:
            signal.signal(sig, _shutdown)
        except (ValueError, OSError):  # pragma: no cover - non-main thread
            pass
