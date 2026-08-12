from __future__ import annotations

import contextlib
import io
import os
import unittest
from unittest import mock

from xiaohei_plugin_sdk.noop import NoopAutomationPlugin
from xiaohei_plugin_sdk.server import _load_plugin, _main, run_local_stub, serve


def _grpc_available() -> bool:
    try:
        import grpc  # noqa: F401
        import plugin.v1  # noqa: F401

        return True
    except Exception:  # noqa: BLE001
        return False


class ServeStartsTransportTest(unittest.TestCase):
    """Round-5: the Python SDK is a real go-plugin transport, not a print-and-exit stub."""

    @unittest.skipUnless(_grpc_available(), "grpcio not installed")
    def test_serve_block_false_binds_socket_and_prints_handshake(self) -> None:
        import grpc

        from plugin.v1 import core_pb2_grpc as c_grpc
        from plugin.v1 import types_pb2 as t_pb2

        buf = io.StringIO()
        with contextlib.redirect_stdout(buf):
            handle = serve(NoopAutomationPlugin(), block=False, allow_dev=True)
        try:
            line = buf.getvalue().strip()
            # 1|1|unix|<addr>|grpc|
            parts = line.split("|")
            self.assertEqual(parts[0], "1", "core protocol version")
            self.assertEqual(parts[1], "1", "app protocol version")
            self.assertEqual(parts[2], "unix", "network")
            self.assertEqual(parts[4], "grpc", "protocol type")
            self.assertTrue(os.path.exists(handle.addr), "unix socket exists")

            # A real gRPC client can reach the CoreService over the socket.
            channel = grpc.insecure_channel(f"unix:{handle.addr}")
            stub = c_grpc.CoreServiceStub(channel)
            manifest = stub.GetManifest(t_pb2.Empty())
            self.assertEqual(manifest.plugin_id, "python_noop")
            channel.close()
        finally:
            handle.stop()

    def test_run_local_stub_starts_transport(self) -> None:
        # run_local_stub is a thin wrapper over serve(); it must NOT exit.
        with mock.patch("xiaohei_plugin_sdk.server.serve") as serve_mock:
            out = io.StringIO()
            with contextlib.redirect_stdout(out):
                result = run_local_stub(NoopAutomationPlugin())
            serve_mock.assert_called_once()
            self.assertIn("transport", out.getvalue().lower())
            self.assertIsNone(result)

    @unittest.skipUnless(_grpc_available(), "grpcio not installed")
    def test_production_launch_without_client_cert_fails_closed(self) -> None:
        with self.assertRaisesRegex(Exception, "PLUGIN_CLIENT_CERT"):
            serve(
                NoopAutomationPlugin(),
                block=False,
                environ={"XIAOHEI_PLUGIN": "xiaoheiplay"},
            )

    @unittest.skipUnless(_grpc_available(), "grpcio not installed")
    def test_production_launch_with_malformed_client_cert_fails_closed(self) -> None:
        with self.assertRaisesRegex(Exception, "malformed PLUGIN_CLIENT_CERT"):
            serve(
                NoopAutomationPlugin(),
                block=False,
                environ={
                    "XIAOHEI_PLUGIN": "xiaoheiplay",
                    "PLUGIN_CLIENT_CERT": "not-a-certificate",
                },
            )


class LoadPluginTest(unittest.TestCase):
    def test_load_plugin_resolves_module_attribute(self) -> None:
        plugin = _load_plugin("xiaohei_plugin_sdk.noop:NoopAutomationPlugin")
        self.assertIsInstance(plugin, NoopAutomationPlugin)

    def test_load_plugin_factory_callable(self) -> None:
        plugin = _load_plugin("examples.echo_plugin:build_plugin")
        from xiaohei_plugin_sdk.interfaces import CoreService

        self.assertIsInstance(plugin, CoreService)

    def test_load_plugin_requires_colon(self) -> None:
        with self.assertRaises(SystemExit):
            _load_plugin("module.only")


class MainTest(unittest.TestCase):
    def test_main_without_host_cookie_exits_nonzero(self) -> None:
        # No host cookie + not --dev -> return code 2 (go-plugin diagnostic path).
        with mock.patch.dict(os.environ, {}, clear=True):
            rc = _main(["--plugin-impl", "xiaohei_plugin_sdk.noop:NoopAutomationPlugin"])
        self.assertEqual(rc, 2)


if __name__ == "__main__":
    unittest.main()
