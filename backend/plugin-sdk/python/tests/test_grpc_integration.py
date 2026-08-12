from __future__ import annotations

import contextlib
import io
import unittest


def _grpc_available() -> bool:
    try:
        import grpc  # noqa: F401
        import plugin.v1  # noqa: F401

        return True
    except Exception:  # noqa: BLE001
        return False


@unittest.skipUnless(_grpc_available(), "grpcio not installed")
class GrpcIntegrationTest(unittest.TestCase):
    """Round-5: real gRPC round-trip against the go-plugin transport services."""

    @classmethod
    def setUpClass(cls) -> None:
        import grpc

        from plugin.v1 import automation_pb2_grpc as a_grpc
        from plugin.v1 import core_pb2_grpc as c_grpc

        from examples.echo_plugin import EchoPlugin
        from xiaohei_plugin_sdk.server import serve

        cls._grpc = grpc
        cls._out = io.StringIO()
        cls._stdout_ctx = contextlib.redirect_stdout(cls._out)
        cls._stdout_ctx.__enter__()
        cls._handle = serve(EchoPlugin(), block=False, allow_dev=True)
        cls._channel = grpc.insecure_channel(f"unix:{cls._handle.addr}")
        cls.core = c_grpc.CoreServiceStub(cls._channel)
        cls.auto = a_grpc.AutomationServiceStub(cls._channel)

    @classmethod
    def tearDownClass(cls) -> None:
        cls._channel.close()
        cls._handle.stop()
        cls._stdout_ctx.__exit__(None, None, None)

    def test_handshake_line_on_stdout(self) -> None:
        line = self._out.getvalue().strip()
        parts = line.split("|")
        self.assertEqual(parts[0], "1")
        self.assertEqual(parts[1], "1")
        self.assertEqual(parts[2], "unix")
        self.assertEqual(parts[4], "grpc")

    def test_core_manifest_round_trip(self) -> None:
        from plugin.v1 import types_pb2 as t_pb2

        manifest = self.core.GetManifest(t_pb2.Empty())
        self.assertEqual(manifest.plugin_id, "python_echo")
        # Automation features map to proto enum values (canonical names).
        self.assertIn(1, [f for f in manifest.automation.features])
        self.assertIn(2, [f for f in manifest.automation.features])

    def test_core_health_round_trip(self) -> None:
        from plugin.v1 import types_pb2 as t_pb2

        health = self.core.Health(t_pb2.HealthCheckRequest(instance_id="i-1"))
        self.assertEqual(health.status, t_pb2.HEALTH_STATUS_OK)
        self.assertGreater(health.unix_millis, 0)

    def test_core_config_schema_round_trip(self) -> None:
        from plugin.v1 import types_pb2 as t_pb2

        schema = self.core.GetConfigSchema(t_pb2.Empty())
        self.assertIn("base_url", schema.json_schema)

    def test_automation_lifecycle_round_trip(self) -> None:
        from plugin.v1 import automation_pb2 as a_pb2
        from plugin.v1 import types_pb2 as t_pb2

        areas = self.auto.ListAreas(t_pb2.Empty())
        self.assertEqual(areas.items[0].name, "area-1")

        lines = self.auto.ListLines(t_pb2.Empty())
        self.assertEqual(lines.items[0].area_id, 1)
        self.assertEqual(lines.items[0].name, "line-10")

        created = self.auto.CreateInstance(
            a_pb2.CreateInstanceRequest(line_id=10, name="vps-echo", password="secret")
        )
        self.assertGreater(created.instance_id, 0)

        started = self.auto.Start(a_pb2.StartRequest(instance_id=created.instance_id))
        self.assertTrue(started.ok)
        self.assertEqual(started.error_code, "")

    def test_automation_catalog_round_trip(self) -> None:
        from plugin.v1 import automation_pb2 as a_pb2
        from plugin.v1 import types_pb2 as t_pb2

        pkgs = self.auto.ListPackages(a_pb2.ListPackagesRequest(line_id=10))
        self.assertEqual(pkgs.items[0].name, "pkg-100")
        self.assertEqual(pkgs.items[0].cpu, 2)

        imgs = self.auto.ListImages(a_pb2.ListImagesRequest(line_id=10))
        self.assertEqual(imgs.items[0].name, "debian-12")

    def test_error_mapping_aborts_internal(self) -> None:
        from xiaohei_plugin_sdk.noop import NoopAutomationPlugin
        from xiaohei_plugin_sdk.server import serve as _serve

        class _BoomPlugin(NoopAutomationPlugin):
            def health(self, instance_id: str):
                raise RuntimeError("internal path /etc/passwd leaked here")

        handle = _serve(_BoomPlugin(), block=False, allow_dev=True)
        try:
            channel = self._grpc.insecure_channel(f"unix:{handle.addr}")
            from plugin.v1 import core_pb2_grpc as c_grpc
            from plugin.v1 import types_pb2 as t_pb2

            stub = c_grpc.CoreServiceStub(channel)
            with self.assertRaises(self._grpc.RpcError) as ctx:
                stub.Health(t_pb2.HealthCheckRequest(instance_id="x"))
            self.assertEqual(ctx.exception.code(), self._grpc.StatusCode.INTERNAL)
            # The raw message (containing a filesystem path) must NOT leak.
            self.assertNotIn("/etc/passwd", ctx.exception.details())
            channel.close()
        finally:
            handle.stop()


if __name__ == "__main__":
    unittest.main()
