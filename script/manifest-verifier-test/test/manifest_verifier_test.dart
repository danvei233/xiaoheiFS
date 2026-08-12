import 'dart:convert';
import 'dart:typed_data';

import 'package:cryptography/cryptography.dart';
import 'package:test/test.dart';

// Import the real verifier + replay guard source from both Flutter apps.
// Relative imports keep this a pure-Dart test (no Flutter SDK required).
import '../../../app/xiaoheifs_userapp/lib/core/update/manifest_verifier.dart'
    as user_verifier;
import '../../../app/xiaoheifs_app/lib/services/manifest_verifier.dart'
    as admin_verifier;
import '../../../app/xiaoheifs_userapp/lib/core/update/update_replay_guard.dart'
    as replay;

const String testPackageName = 'com.example.app';

/// Deterministic 32-byte Ed25519 seeds for test-only signing keys.
/// epoch 1 -> seed bytes 0..31, epoch 2 -> seed bytes 32..63.
Uint8List _seedForEpoch(int epoch) =>
    Uint8List.fromList(List<int>.generate(32, (i) => i + (epoch - 1) * 32));

Future<SimpleKeyPair> _keyPair(int epoch) =>
    Ed25519().newKeyPairFromSeed(_seedForEpoch(epoch));

Future<Uint8List> _publicKey(int epoch) async => Uint8List.fromList(
  (await (await _keyPair(epoch)).extractPublicKey()).bytes,
);

Future<Uint8List> _sign(int epoch, String canonical) async {
  final sig = await Ed25519().sign(
    utf8.encode(canonical),
    keyPair: await _keyPair(epoch),
  );
  return Uint8List.fromList(sig.bytes);
}

String hexBytes(List<int> bytes) {
  final buf = StringBuffer();
  for (final b in bytes) {
    buf.write(b.toRadixString(16).padLeft(2, '0'));
  }
  return buf.toString();
}

/// Builds a manifest whose signature covers the canonical payload of the
/// fields it carries, using the key for [keyEpoch].
Future<Map<String, dynamic>> signedManifest({
  required String apkSha256,
  required int latestVersionCode,
  required int issuedAt,
  String? packageName,
  bool hasUpdate = true,
  int keyEpoch = 1,
  String latestVersion = '1.2.3',
  bool forceUpdate = false,
  String apkUrl = 'https://updates.example.com/app.apk',
  String changelog = 'bug fixes',
}) async {
  final canonical = user_verifier.ManifestVerifier.buildCanonicalPayload(
    packageName: packageName ?? testPackageName,
    hasUpdate: hasUpdate,
    keyEpoch: keyEpoch,
    latestVersion: latestVersion,
    latestVersionCode: latestVersionCode,
    forceUpdate: forceUpdate,
    apkUrl: apkUrl,
    apkSha256: apkSha256,
    changelog: changelog,
    issuedAt: issuedAt,
  );
  final sig = await _sign(keyEpoch, canonical);
  return {
    'hasUpdate': hasUpdate,
    'keyEpoch': keyEpoch,
    'latestVersion': latestVersion,
    'latestVersionCode': latestVersionCode,
    'forceUpdate': forceUpdate,
    'apkUrl': apkUrl,
    'sha256': apkSha256,
    'changelog': changelog,
    'issuedAt': issuedAt,
    'algorithm': 'ed25519',
    'signature': base64Encode(sig),
  };
}

typedef ManifestCheck = Future<bool> Function(Map<String, dynamic> manifest);

void main() {
  final now = DateTime.fromMillisecondsSinceEpoch(1754952800000, isUtc: true);
  final freshIssuedAt = now.millisecondsSinceEpoch ~/ 1000;
  const apkSha256 =
      'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';
  late Map<int, String> pubKeys;

  setUpAll(() async {
    pubKeys = {
      1: hexBytes(await _publicKey(1)),
      2: hexBytes(await _publicKey(2)),
    };
  });

  final verifiers = <String, ManifestCheck>{
    'userapp': (m) => user_verifier.ManifestVerifier.verifyManifest(
      manifest: m,
      packageName: testPackageName,
      currentVersionCode: 3,
      publicKeys: pubKeys,
      now: now,
    ),
    'adminapp': (m) => admin_verifier.ManifestVerifier.verifyManifest(
      manifest: m,
      packageName: testPackageName,
      currentVersionCode: 3,
      publicKeys: pubKeys,
      now: now,
    ),
  };

  for (final entry in verifiers.entries) {
    group('ManifestVerifier.verifyManifest (${entry.key})', () {
      ManifestCheck verify = entry.value;

      test('accepts a correctly signed manifest', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
        );
        expect(await verify(m), isTrue);
      });

      test('rejects a tampered artifact hash (sha256)', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
        );
        m['sha256'] =
            'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb';
        expect(await verify(m), isFalse);
      });

      test('rejects a tampered URL', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
        );
        m['apkUrl'] = 'https://evil.example.com/trojan.apk';
        expect(await verify(m), isFalse);
      });

      test('rejects a tampered version string', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
          latestVersion: '1.2.3',
        );
        m['latestVersion'] = '9.9.9';
        expect(await verify(m), isFalse);
      });

      test('rejects a tampered versionCode', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
        );
        m['latestVersionCode'] = 5;
        expect(await verify(m), isFalse);
      });

      test('rejects a tampered forceUpdate', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
          forceUpdate: false,
        );
        m['forceUpdate'] = true;
        expect(await verify(m), isFalse);
      });

      test(
        'rejects a tampered hasUpdate (update decision cannot be flipped)',
        () async {
          final m = await signedManifest(
            apkSha256: apkSha256,
            latestVersionCode: 4,
            issuedAt: freshIssuedAt,
            hasUpdate: true,
          );
          m['hasUpdate'] = false;
          // hasUpdate is part of the signed payload: flipping it invalidates the
          // signature, so the client must not trust the flipped decision.
          expect(await verify(m), isFalse);
        },
      );

      test('rejects a tampered keyEpoch', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
          keyEpoch: 1,
        );
        m['keyEpoch'] = 2;
        expect(await verify(m), isFalse);
      });

      test('rejects a signature made with a different key', () async {
        // Sign with epoch 2 but claim epoch 1: the key lookup for epoch 1 fails
        // to verify the signature made with the epoch-2 key.
        final canonical = user_verifier.ManifestVerifier.buildCanonicalPayload(
          packageName: testPackageName,
          hasUpdate: true,
          keyEpoch: 1,
          latestVersion: '1.2.3',
          latestVersionCode: 4,
          forceUpdate: false,
          apkUrl: 'https://updates.example.com/app.apk',
          apkSha256: apkSha256,
          changelog: 'bug fixes',
          issuedAt: freshIssuedAt,
        );
        final sig = await _sign(2, canonical);
        final m = {
          'hasUpdate': true,
          'keyEpoch': 1,
          'latestVersion': '1.2.3',
          'latestVersionCode': 4,
          'forceUpdate': false,
          'apkUrl': 'https://updates.example.com/app.apk',
          'sha256': apkSha256,
          'changelog': 'bug fixes',
          'issuedAt': freshIssuedAt,
          'algorithm': 'ed25519',
          'signature': base64Encode(sig),
        };
        expect(await verify(m), isFalse);
      });

      test('rejects a manifest with no signature', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
        );
        m.remove('signature');
        expect(await verify(m), isFalse);
      });

      test('rejects a malformed (non-base64) signature', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
        );
        m['signature'] = '!!!not-base64!!!';
        expect(await verify(m), isFalse);
      });

      test('rejects a signature of the wrong length', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
        );
        m['signature'] = base64Encode(List<int>.filled(63, 1));
        expect(await verify(m), isFalse);
      });

      test('rejects an unsupported algorithm', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
        );
        m['algorithm'] = 'rsa-sha256';
        expect(await verify(m), isFalse);
      });

      test('rejects a missing keyEpoch', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
        );
        m.remove('keyEpoch');
        expect(await verify(m), isFalse);
      });

      test('rejects an unknown keyEpoch', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
          keyEpoch: 99, // signed, but no key embedded for epoch 99
        );
        expect(await verify(m), isFalse);
      });

      test('rejects a missing/placeholder public key', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
        );
        expect(await verify(m), isTrue); // correct key
        // Placeholder key must fail closed.
        expect(
          await user_verifier.ManifestVerifier.verifyManifest(
            manifest: m,
            packageName: testPackageName,
            currentVersionCode: 3,
            publicKeys: {
              1: user_verifier.ManifestVerifier.placeholderPublicKeyHex,
            },
            now: now,
          ),
          isFalse,
        );
        // Malformed key (not 64 hex chars) must fail closed.
        expect(
          await user_verifier.ManifestVerifier.verifyManifest(
            manifest: m,
            packageName: testPackageName,
            currentVersionCode: 3,
            publicKeys: {1: 'deadbeef'},
            now: now,
          ),
          isFalse,
        );
      });

      test('rejects a non-https download URL even when signed', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
          apkUrl: 'http://updates.example.com/app.apk',
        );
        expect(await verify(m), isFalse);
      });

      test('rejects an expired manifest (replay protection)', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt - 8 * 24 * 3600, // 8 days old
        );
        expect(await verify(m), isFalse);
      });

      test('rejects a manifest from the future (clock skew)', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt + 3600, // 1 hour ahead
        );
        expect(await verify(m), isFalse);
      });

      test('rejects a huge issuedAt that would wrap in int64*1000', () async {
        // issuedAt ≈ 2^64/1000: naive issuedAt*1000 wraps in a 64-bit VM to a
        // value inside the replay window, bypassing the age check. The verifier
        // must reject it before the multiplication.
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: 18446744150000000,
        );
        expect(
          await verify(m),
          isFalse,
          reason: 'issuedAt*1000 overflow must not slip past the replay window',
        );
      });

      test(
        'accepts a manifest at the age boundary, rejects just past it',
        () async {
          // Exactly 7 days old is still acceptable.
          final edge = await signedManifest(
            apkSha256: apkSha256,
            latestVersionCode: 4,
            issuedAt: freshIssuedAt - 7 * 24 * 3600,
          );
          expect(await verify(edge), isTrue);
          // One second older is rejected.
          final past = await signedManifest(
            apkSha256: apkSha256,
            latestVersionCode: 4,
            issuedAt: freshIssuedAt - 7 * 24 * 3600 - 1,
          );
          expect(await verify(past), isFalse);
          // Exactly 5 minutes ahead is acceptable (clock-skew tolerance) ...
          final ahead = await signedManifest(
            apkSha256: apkSha256,
            latestVersionCode: 4,
            issuedAt: freshIssuedAt + 5 * 60,
          );
          expect(await verify(ahead), isTrue);
          // ... just past it is rejected.
          final tooFar = await signedManifest(
            apkSha256: apkSha256,
            latestVersionCode: 4,
            issuedAt: freshIssuedAt + 5 * 60 + 1,
          );
          expect(await verify(tooFar), isFalse);
        },
      );

      test('rejects a downgrade (signed versionCode <= current)', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 3, // not higher than current (3)
          issuedAt: freshIssuedAt,
        );
        expect(await verify(m), isFalse);
      });

      test(
        'rejects when the manifest is bound to a different package',
        () async {
          final m = await signedManifest(
            apkSha256: apkSha256,
            latestVersionCode: 4,
            issuedAt: freshIssuedAt,
            packageName: 'com.evil.other',
          );
          expect(await verify(m), isFalse);
        },
      );

      test('rejects an APK hash that is not lowercase hex', () async {
        final m = await signedManifest(
          apkSha256: apkSha256.toUpperCase(),
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
        );
        expect(await verify(m), isFalse);
      });

      test('accepts a manifest signed with the rotated key when both keys are '
          'embedded (legal dual-key rotation)', () async {
        final m = await signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
          keyEpoch: 2,
        );
        expect(await verify(m), isTrue);
      });
    });
  }

  group('ManifestReplayGuard (cross-session replay)', () {
    test('accepts first manifest, rejects replay and downgrade', () async {
      final store = replay.InMemoryManifestStateStore();
      final guard = replay.ManifestReplayGuard(store);
      const pkg = testPackageName;
      const epoch = 1;

      expect(
        await guard.isAcceptable(
          packageName: pkg,
          keyEpoch: epoch,
          versionCode: 4,
          issuedAt: 1754952800,
        ),
        isTrue,
      );
      await guard.recordAccepted(
        packageName: pkg,
        keyEpoch: epoch,
        versionCode: 4,
        issuedAt: 1754952800,
      );

      // Exact replay of the same signed manifest is rejected.
      expect(
        await guard.isAcceptable(
          packageName: pkg,
          keyEpoch: epoch,
          versionCode: 4,
          issuedAt: 1754952800,
        ),
        isFalse,
      );
      // Same version, older issuedAt is rejected.
      expect(
        await guard.isAcceptable(
          packageName: pkg,
          keyEpoch: epoch,
          versionCode: 4,
          issuedAt: 1754952799,
        ),
        isFalse,
      );
      // Older version, even with newer issuedAt, is a downgrade: rejected.
      expect(
        await guard.isAcceptable(
          packageName: pkg,
          keyEpoch: epoch,
          versionCode: 3,
          issuedAt: 1754952801,
        ),
        isFalse,
      );
      // A strictly newer manifest is accepted and advances the record.
      expect(
        await guard.isAcceptable(
          packageName: pkg,
          keyEpoch: epoch,
          versionCode: 5,
          issuedAt: 1754953000,
        ),
        isTrue,
      );
      await guard.recordAccepted(
        packageName: pkg,
        keyEpoch: epoch,
        versionCode: 5,
        issuedAt: 1754953000,
      );
      // After the advance, the previously accepted record is now stale.
      expect(
        await guard.isAcceptable(
          packageName: pkg,
          keyEpoch: epoch,
          versionCode: 4,
          issuedAt: 1754952800,
        ),
        isFalse,
      );
    });

    test(
      'offering without consent does not suppress the update (F14): a checked '
      'manifest stays acceptable until the user consents',
      () async {
        // F14 regression: the client must NOT record "accepted" just because a
        // manifest was offered. A user who dismisses a non-forced update is
        // re-prompted next launch; only an explicit recordAcceptedConsent write
        // makes the same manifest a replay.
        final store = replay.InMemoryManifestStateStore();
        final guard = replay.ManifestReplayGuard(store);

        // Offered (checked) but never consented: still acceptable next launch.
        expect(
          await guard.isAcceptable(
            packageName: 'com.example.app',
            keyEpoch: 1,
            versionCode: 4,
            issuedAt: 1754952800,
          ),
          isTrue,
        );
        expect(
          await guard.isAcceptable(
            packageName: 'com.example.app',
            keyEpoch: 1,
            versionCode: 4,
            issuedAt: 1754952800,
          ),
          isTrue,
        );

        // After the user consents (recordAcceptedConsent -> recordAccepted),
        // the same manifest is a replay and must be rejected.
        await guard.recordAccepted(
          packageName: 'com.example.app',
          keyEpoch: 1,
          versionCode: 4,
          issuedAt: 1754952800,
        );
        expect(
          await guard.isAcceptable(
            packageName: 'com.example.app',
            keyEpoch: 1,
            versionCode: 4,
            issuedAt: 1754952800,
          ),
          isFalse,
        );
      },
    );

    test('keeps records separate per package and per keyEpoch', () async {
      final store = replay.InMemoryManifestStateStore();
      final guard = replay.ManifestReplayGuard(store);

      await guard.recordAccepted(
        packageName: 'com.example.app',
        keyEpoch: 1,
        versionCode: 9,
        issuedAt: 1754952800,
      );
      // Different package: independent.
      expect(
        await guard.isAcceptable(
          packageName: 'com.example.other',
          keyEpoch: 1,
          versionCode: 9,
          issuedAt: 1754952800,
        ),
        isTrue,
      );
      // Same package, different epoch (key rotation): independent.
      expect(
        await guard.isAcceptable(
          packageName: 'com.example.app',
          keyEpoch: 2,
          versionCode: 1,
          issuedAt: 1754952800,
        ),
        isTrue,
      );
    });
  });

  group('golden vectors (shared with the Go signer)', () {
    test('matches the pinned Go signer public key and signature', () async {
      // The Go signer test (script/update-sign/canonical_test.go) pins the
      // same seed (bytes 0..31), package and manifest and logs the exact
      // public key + signature. Both implementations MUST agree here so a
      // signature produced by update-sign is accepted by the clients.
      const expectedPubHex =
          '03a107bff3ce10be1d70dd18e74bc09967e4d6309ba50d5f1ddc8664125531b8';
      final pubHex = hexBytes(await _publicKey(1));
      expect(pubHex, expectedPubHex);

      final m = await signedManifest(
        apkSha256: apkSha256,
        latestVersionCode: 4,
        issuedAt: 1754952800,
        latestVersion: '1.2.3',
        apkUrl: 'https://updates.example.com/app.apk',
      );
      // Verification must succeed with the golden key.
      expect(await verifiers['userapp']!(m), isTrue);
      expect(await verifiers['adminapp']!(m), isTrue);
    });

    test('verifies a signature produced by the Go signer (fixed bytes)', () async {
      // The Go signer (script/update-sign/canonical_test.go: TestGoldenVector)
      // computes this base64 Ed25519 signature over the SAME canonical payload
      // below. Pinning the exact signature means a Go->Dart byte mismatch can
      // never slip past CI: if either side changes the canonical serialization,
      // this verification fails.
      const goSignatureB64 =
          'ikau1RjAMHxlF6UsZKt6UjfeuFMdTFLnSQhGD8+2msb4gyH5WKy2kv9WWmiKwtFD/DIoMVw/3HizdHG4LyeCDw==';
      final m = <String, dynamic>{
        'hasUpdate': true,
        'keyEpoch': 1,
        'latestVersion': '1.2.3',
        'latestVersionCode': 4,
        'forceUpdate': false,
        'apkUrl': 'https://updates.example.com/app.apk',
        'sha256': apkSha256,
        'changelog': 'bug fixes',
        'issuedAt': 1754952800,
        'algorithm': 'ed25519',
        'signature': goSignatureB64,
      };
      expect(
        await verifiers['userapp']!(m),
        isTrue,
        reason: 'userapp must accept the Go signer golden signature',
      );
      expect(
        await verifiers['adminapp']!(m),
        isTrue,
        reason: 'adminapp must accept the Go signer golden signature',
      );

      // Tampering any signed field invalidates the Go-produced signature.
      final tampered = Map<String, dynamic>.from(m)..['latestVersionCode'] = 5;
      expect(await verifiers['userapp']!(tampered), isFalse);
    });
  });

  group('canonical payload determinism', () {
    test('is byte-identical across the two verifier implementations', () {
      final a = user_verifier.ManifestVerifier.buildCanonicalPayload(
        packageName: 'com.example.app',
        hasUpdate: true,
        keyEpoch: 1,
        latestVersion: '1.2.3',
        latestVersionCode: 4,
        forceUpdate: true,
        apkUrl: 'https://updates.example.com/app.apk',
        apkSha256: apkSha256,
        changelog: 'line1\nline2',
        issuedAt: 1754952800,
      );
      final b = admin_verifier.ManifestVerifier.buildCanonicalPayload(
        packageName: 'com.example.app',
        hasUpdate: true,
        keyEpoch: 1,
        latestVersion: '1.2.3',
        latestVersionCode: 4,
        forceUpdate: true,
        apkUrl: 'https://updates.example.com/app.apk',
        apkSha256: apkSha256,
        changelog: 'line1\nline2',
        issuedAt: 1754952800,
      );
      expect(a, b);
      expect(a, contains('hasUpdate=1'));
      expect(a, contains('epoch=1'));
      expect(a, contains('force=1'));
      expect(a, contains('notes=line1\\nline2'));
      expect(a, contains('issuedAt=1754952800'));
    });

    test('escapes control characters in field values', () {
      final payload = user_verifier.ManifestVerifier.buildCanonicalPayload(
        packageName: 'com.example.app',
        hasUpdate: false,
        keyEpoch: 2,
        latestVersion: '1.0',
        latestVersionCode: 1,
        forceUpdate: false,
        apkUrl: 'https://updates.example.com/a\\b\nc',
        apkSha256: apkSha256,
        changelog: 'tab\there',
        issuedAt: 1,
      );
      expect(payload, contains('hasUpdate=0'));
      expect(payload, contains('epoch=2'));
      expect(payload, contains(r'url=https://updates.example.com/a\\b\nc'));
      expect(payload, contains(r'notes=tab\there'));
    });
  });
}
