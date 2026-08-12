from __future__ import annotations

import os
import tempfile
import unittest

from xiaohei_plugin_sdk import transport


class HandshakeParseTest(unittest.TestCase):
    def test_parses_host_environment(self) -> None:
        env = {
            "XIAOHEI_PLUGIN": "xiaoheiplay",
            "PLUGIN_PROTOCOL_VERSIONS": "1",
            "PLUGIN_MIN_PORT": "10000",
            "PLUGIN_MAX_PORT": "25000",
            "PLUGIN_CLIENT_CERT": "client cert",
            "PLUGIN_UNIX_SOCKET_DIR": "/tmp/plugin-dir",
        }
        parsed = transport.parse_env(env)
        self.assertEqual(parsed.magic_cookie_key, "XIAOHEI_PLUGIN")
        self.assertEqual(parsed.magic_cookie_value, "xiaoheiplay")
        self.assertEqual(parsed.protocol_versions, (1,))
        self.assertEqual(parsed.min_port, 10000)
        self.assertEqual(parsed.max_port, 25000)
        self.assertEqual(parsed.client_cert_pem, "client cert")
        self.assertEqual(parsed.unix_socket_dir, "/tmp/plugin-dir")

    def test_defaults_without_host_env(self) -> None:
        # No host cookie -> empty, so validate_handshake can reject a direct run.
        parsed = transport.parse_env({})
        self.assertEqual(parsed.magic_cookie_key, "")
        self.assertEqual(parsed.magic_cookie_value, "")
        self.assertEqual(parsed.protocol_versions, ())
        self.assertIsNone(parsed.min_port)
        self.assertEqual(parsed.client_cert_pem, "")
        self.assertIsNone(parsed.unix_socket_dir)

    def test_zero_ports_mean_unset(self) -> None:
        parsed = transport.parse_env({"PLUGIN_MIN_PORT": "0", "PLUGIN_MAX_PORT": "0"})
        self.assertIsNone(parsed.min_port)
        self.assertIsNone(parsed.max_port)

    def test_malformed_ports_rejected(self) -> None:
        with self.assertRaises(transport.HandshakeError):
            transport.parse_env({"PLUGIN_MIN_PORT": "abc"})
        with self.assertRaises(transport.HandshakeError):
            transport.parse_env({"PLUGIN_MAX_PORT": "-1"})

    def test_malformed_protocol_versions_rejected(self) -> None:
        with self.assertRaises(transport.HandshakeError):
            transport.parse_env({"PLUGIN_PROTOCOL_VERSIONS": "1,notanint"})


class HandshakeValidationTest(unittest.TestCase):
    def test_missing_cookie_rejected_unless_dev(self) -> None:
        with self.assertRaises(transport.HandshakeError):
            transport.validate_handshake({}, allow_dev=False)
        # allow_dev returns the (empty-cookie) env instead of raising.
        env = transport.validate_handshake({}, allow_dev=True)
        self.assertEqual(env.magic_cookie_value, "")

    def test_wrong_cookie_rejected_even_in_dev(self) -> None:
        env = {
            "XIAOHEI_PLUGIN": "wrong-value",
        }
        with self.assertRaises(transport.HandshakeError):
            transport.validate_handshake(env, allow_dev=True)

    def test_correct_cookie_accepted(self) -> None:
        env = {
            "XIAOHEI_PLUGIN": "xiaoheiplay",
        }
        parsed = transport.validate_handshake(env)
        self.assertEqual(parsed.magic_cookie_value, "xiaoheiplay")


class HandshakeLineTest(unittest.TestCase):
    def test_build_line_format(self) -> None:
        line = transport.build_handshake_line("/tmp/xiaohei-plugin-1.sock")
        parts = line.split("|")
        self.assertEqual(parts[0], "1", "core protocol version")
        self.assertEqual(parts[1], "1", "app protocol version")
        self.assertEqual(parts[2], "unix")
        self.assertEqual(parts[3], "/tmp/xiaohei-plugin-1.sock")
        self.assertEqual(parts[4], "grpc")
        self.assertEqual(parts[5], "", "dev mode has no mTLS cert")

    def test_build_line_includes_auto_mtls_certificate(self) -> None:
        line = transport.build_handshake_line("/tmp/plugin.sock", server_cert="raw-der-base64")
        self.assertEqual(line.split("|")[5], "raw-der-base64")

    def test_build_line_tcp(self) -> None:
        line = transport.build_handshake_line("127.0.0.1:12345", network="tcp")
        parts = line.split("|")
        self.assertEqual(parts[2], "tcp")
        self.assertEqual(parts[3], "127.0.0.1:12345")
        self.assertEqual(parts[4], "grpc")


class UnixSocketPathTest(unittest.TestCase):
    def test_uses_host_private_socket_directory(self) -> None:
        with tempfile.TemporaryDirectory() as socket_dir:
            env = transport.parse_env({"PLUGIN_UNIX_SOCKET_DIR": socket_dir})
            path = transport._default_socket_path(env)
            self.assertEqual(os.path.dirname(path), socket_dir)

    def test_rejects_relative_host_socket_directory(self) -> None:
        env = transport.parse_env({"PLUGIN_UNIX_SOCKET_DIR": "relative/plugin-dir"})
        with self.assertRaises(transport.HandshakeError):
            transport._default_socket_path(env)


if __name__ == "__main__":
    unittest.main()
