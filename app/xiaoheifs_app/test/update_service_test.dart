import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:crypto/crypto.dart';
import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:opensource_adminapp/services/manifest_verifier.dart';
import 'package:opensource_adminapp/services/update_config.dart';
import 'package:opensource_adminapp/services/update_replay_guard.dart';
import 'package:opensource_adminapp/services/update_service.dart';

const String testPackageName = 'com.example.app';

final Uint8List _seed = Uint8List.fromList(List<int>.generate(32, (i) => i));

Future<SimpleKeyPair> _keyPair() => Ed25519().newKeyPairFromSeed(_seed);

Future<Uint8List> _publicKey() async =>
    Uint8List.fromList((await (await _keyPair()).extractPublicKey()).bytes);

String hexBytes(List<int> bytes) {
  final buf = StringBuffer();
  for (final b in bytes) {
    buf.write(b.toRadixString(16).padLeft(2, '0'));
  }
  return buf.toString();
}

Future<Map<String, dynamic>> _signedManifest({
  required String apkSha256,
  required int latestVersionCode,
  required int issuedAt,
  bool hasUpdate = true,
  int keyEpoch = 1,
  String latestVersion = '1.2.3',
  bool forceUpdate = false,
  String apkUrl = 'https://updates.example.com/app-v1.2.3.apk',
  String changelog = 'bug fixes',
}) async {
  final canonical = ManifestVerifier.buildCanonicalPayload(
    packageName: testPackageName,
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
  final sig = await Ed25519().sign(
    utf8.encode(canonical),
    keyPair: await _keyPair(),
  );
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
    'signature': base64Encode(sig.bytes),
  };
}

void main() {
  final now = DateTime.fromMillisecondsSinceEpoch(1754952800000, isUtc: true);
  final freshIssuedAt = now.millisecondsSinceEpoch ~/ 1000;
  // 真实 APK 字节与其 sha256：受控下载路径用真实字节校验，替代旧的占位 hash。
  final apkBytes = Uint8List.fromList(List<int>.generate(256, (i) => i % 251));
  final apkSha256 = sha256.convert(apkBytes).toString();
  late Map<int, String> pubKeys;

  setUpAll(() async {
    pubKeys = {1: hexBytes(await _publicKey())};
  });

  setUp(() {
    UpdateConfig.serverBaseUrl = 'https://updates.example.com';
  });

  UpdateService serviceWith(
    Map<String, dynamic> payload, {
    ManifestStateStore? store,
    Future<bool> Function(Uri url)? launcher,
    Uint8List? downloadBytes,
  }) {
    return UpdateService(
      client: MockClient((req) async {
        // .apk 下载路径返回真实 APK 字节（供 downloadAndVerifyApk 校验）；
        // 其余路径返回更新清单 JSON。
        if (req.url.path.toLowerCase().endsWith('.apk')) {
          return http.Response.bytes(downloadBytes ?? apkBytes, 200);
        }
        return http.Response(jsonEncode(payload), 200);
      }),
      publicKeys: pubKeys,
      clock: () => now,
      stateStore: store ?? InMemoryManifestStateStore(),
      launcher: launcher ?? (uri) async => true,
    );
  }

  group('UpdateService.checkForUpdate', () {
    test('accepts an update signed by the trusted key', () async {
      final svc = serviceWith(
        await _signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
        ),
      );
      final info = await svc.checkForUpdate(
        packageName: testPackageName,
        versionCode: 3,
      );
      expect(info, isNotNull);
      expect(info!.apkUrl, 'https://updates.example.com/app-v1.2.3.apk');
      expect(info.apkSha256, apkSha256);
    });

    test('rejects an unsigned update manifest (fail closed)', () async {
      final payload = await _signedManifest(
        apkSha256: apkSha256,
        latestVersionCode: 4,
        issuedAt: freshIssuedAt,
      );
      payload.remove('signature');
      final svc = serviceWith(payload);
      final info = await svc.checkForUpdate(
        packageName: testPackageName,
        versionCode: 3,
      );
      expect(info, isNull);
    });

    test('rejects an update whose artifact hash was tampered', () async {
      final payload = await _signedManifest(
        apkSha256: apkSha256,
        latestVersionCode: 4,
        issuedAt: freshIssuedAt,
      );
      payload['sha256'] =
          'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb';
      final svc = serviceWith(payload);
      final info = await svc.checkForUpdate(
        packageName: testPackageName,
        versionCode: 3,
      );
      expect(info, isNull);
    });

    test('rejects an update whose URL was tampered', () async {
      final payload = await _signedManifest(
        apkSha256: apkSha256,
        latestVersionCode: 4,
        issuedAt: freshIssuedAt,
      );
      // Same host so the host check passes; the signature over the original
      // URL must fail.
      payload['apkUrl'] = 'https://updates.example.com/evil.apk';
      final svc = serviceWith(payload);
      final info = await svc.checkForUpdate(
        packageName: testPackageName,
        versionCode: 3,
      );
      expect(info, isNull);
    });

    test('rejects an update signed with the wrong key', () async {
      final otherSeed = Uint8List.fromList(
        List<int>.generate(32, (i) => i + 1),
      );
      final canonical = ManifestVerifier.buildCanonicalPayload(
        packageName: testPackageName,
        hasUpdate: true,
        keyEpoch: 1,
        latestVersion: '1.2.3',
        latestVersionCode: 4,
        forceUpdate: false,
        apkUrl: 'https://updates.example.com/app-v1.2.3.apk',
        apkSha256: apkSha256,
        changelog: 'bug fixes',
        issuedAt: freshIssuedAt,
      );
      final sig = await Ed25519().sign(
        utf8.encode(canonical),
        keyPair: await Ed25519().newKeyPairFromSeed(otherSeed),
      );
      final payload = {
        'hasUpdate': true,
        'keyEpoch': 1,
        'latestVersion': '1.2.3',
        'latestVersionCode': 4,
        'forceUpdate': false,
        'apkUrl': 'https://updates.example.com/app-v1.2.3.apk',
        'sha256': apkSha256,
        'changelog': 'bug fixes',
        'issuedAt': freshIssuedAt,
        'algorithm': 'ed25519',
        'signature': base64Encode(sig.bytes),
      };
      final svc = serviceWith(payload);
      final info = await svc.checkForUpdate(
        packageName: testPackageName,
        versionCode: 3,
      );
      expect(info, isNull);
    });

    test('re-offers a dismissed (not consented) non-forced update', () async {
      // A user who taps "Later" must see the same update again on the next
      // launch: merely offering/checking must NOT write the replay record.
      final store = InMemoryManifestStateStore();
      final first = await _signedManifest(
        apkSha256: apkSha256,
        latestVersionCode: 4,
        issuedAt: freshIssuedAt,
      );
      final info = await serviceWith(
        first,
        store: store,
      ).checkForUpdate(packageName: testPackageName, versionCode: 3);
      expect(info, isNotNull);

      // Same manifest checked again (next launch, user dismissed last time):
      // still offered because nothing was recorded.
      final again = await serviceWith(
        first,
        store: store,
      ).checkForUpdate(packageName: testPackageName, versionCode: 3);
      expect(again, isNotNull);
    });

    test(
      'rejects a cross-session replay/downgrade after the user consents',
      () async {
        final store = InMemoryManifestStateStore();
        final first = await _signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
        );
        final svc = serviceWith(first, store: store);
        final info = await svc.checkForUpdate(
          packageName: testPackageName,
          versionCode: 3,
        );
        expect(info, isNotNull);

        // The user taps "Update now": only now is the record written.
        await svc.recordAcceptedConsent(info!, packageName: testPackageName);

        // Replaying the same signed manifest later must be rejected.
        final replay = await serviceWith(
          first,
          store: store,
        ).checkForUpdate(packageName: testPackageName, versionCode: 3);
        expect(replay, isNull);

        // A strictly newer signed manifest is still accepted.
        final newer = await _signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 5,
          issuedAt: freshIssuedAt + 60,
        );
        final info2 = await serviceWith(
          newer,
          store: store,
        ).checkForUpdate(packageName: testPackageName, versionCode: 3);
        expect(info2, isNotNull);
      },
    );

    test(
      'rejects a flipped hasUpdate on a signed manifest (decision binding)',
      () async {
        final payload = await _signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
          hasUpdate: true,
        );
        payload['hasUpdate'] = false;
        final svc = serviceWith(payload);
        final info = await svc.checkForUpdate(
          packageName: testPackageName,
          versionCode: 3,
        );
        expect(info, isNull);
      },
    );

    test('rejects a replayed/expired signed manifest', () async {
      final svc = serviceWith(
        await _signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt - 8 * 24 * 3600, // 8 days old
        ),
      );
      final info = await svc.checkForUpdate(
        packageName: testPackageName,
        versionCode: 3,
      );
      expect(info, isNull);
    });

    test('rejects a downgrade signed manifest', () async {
      final svc = serviceWith(
        await _signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 3, // not higher than installed 3
          issuedAt: freshIssuedAt,
        ),
      );
      final info = await svc.checkForUpdate(
        packageName: testPackageName,
        versionCode: 3,
      );
      expect(info, isNull);
    });

    test('rejects an update pointing at a cross-host apkUrl', () async {
      // A tampered manifest must never redirect the admin to an arbitrary host.
      final svc = serviceWith(
        await _signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
          apkUrl: 'https://evil.example.com/trojan.apk',
        ),
      );
      final info = await svc.checkForUpdate(
        packageName: testPackageName,
        versionCode: 3,
      );
      expect(info, isNull);
    });

    test('rejects an update with a non-https apkUrl', () async {
      final svc = serviceWith(
        await _signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
          apkUrl: 'http://updates.example.com/app.apk',
        ),
      );
      final info = await svc.checkForUpdate(
        packageName: testPackageName,
        versionCode: 3,
      );
      expect(info, isNull);
    });

    test('rejects an update with no apkUrl', () async {
      final payload = await _signedManifest(
        apkSha256: apkSha256,
        latestVersionCode: 4,
        issuedAt: freshIssuedAt,
      );
      payload['apkUrl'] = '';
      final svc = serviceWith(payload);
      final info = await svc.checkForUpdate(
        packageName: testPackageName,
        versionCode: 3,
      );
      expect(info, isNull);
    });
  });

  group('UpdateService.openUpdateLink', () {
    test('refuses to open a non-https link', () async {
      final svc = UpdateService();
      expect(
        await svc.openUpdateLink('http://updates.example.com/app.apk'),
        isFalse,
      );
    });

    test('refuses to open a cross-host link', () async {
      final svc = UpdateService();
      expect(
        await svc.openUpdateLink('https://evil.example.com/app.apk'),
        isFalse,
      );
    });
  });

  group('UpdateService.consentAndLaunchUpdate', () {
    test(
      'records consent only AFTER a successful launch (normal update)',
      () async {
        final store = InMemoryManifestStateStore();
        final manifest = await _signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
          forceUpdate: false,
        );
        final svc = serviceWith(manifest, store: store);
        final info = await svc.checkForUpdate(
          packageName: testPackageName,
          versionCode: 3,
        );
        expect(info, isNotNull);

        // Successful handoff: consent is recorded...
        final launched = await svc.consentAndLaunchUpdate(
          info!,
          packageName: testPackageName,
        );
        expect(launched, isTrue);

        // ...so replaying the same signed manifest is rejected.
        final replay = await serviceWith(
          manifest,
          store: store,
        ).checkForUpdate(packageName: testPackageName, versionCode: 3);
        expect(replay, isNull);
      },
    );

    test(
      'does NOT record consent when the launch fails (normal update)',
      () async {
        final store = InMemoryManifestStateStore();
        final manifest = await _signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
          forceUpdate: false,
        );
        final svc = serviceWith(
          manifest,
          store: store,
          launcher: (uri) async => false,
        );
        final info = await svc.checkForUpdate(
          packageName: testPackageName,
          versionCode: 3,
        );
        expect(info, isNotNull);

        // Failed handoff: consent must NOT be recorded...
        final launched = await svc.consentAndLaunchUpdate(
          info!,
          packageName: testPackageName,
        );
        expect(launched, isFalse);

        // ...so the same-version update is still offered next launch.
        final again = await serviceWith(
          manifest,
          store: store,
        ).checkForUpdate(packageName: testPackageName, versionCode: 3);
        expect(again, isNotNull);
      },
    );

    test('does NOT record consent when a FORCED update launch fails', () async {
      final store = InMemoryManifestStateStore();
      final manifest = await _signedManifest(
        apkSha256: apkSha256,
        latestVersionCode: 4,
        issuedAt: freshIssuedAt,
        forceUpdate: true,
      );
      final svc = serviceWith(
        manifest,
        store: store,
        launcher: (uri) async => false,
      );
      final info = await svc.checkForUpdate(
        packageName: testPackageName,
        versionCode: 3,
      );
      expect(info, isNotNull);
      expect(info!.forceUpdate, isTrue);

      // A failed launch of a forced update must not permanently suppress it.
      expect(
        await svc.consentAndLaunchUpdate(info, packageName: testPackageName),
        isFalse,
      );
      final again = await serviceWith(
        manifest,
        store: store,
      ).checkForUpdate(packageName: testPackageName, versionCode: 3);
      expect(again, isNotNull);
    });

    test('records consent when a FORCED update launch succeeds', () async {
      final store = InMemoryManifestStateStore();
      final manifest = await _signedManifest(
        apkSha256: apkSha256,
        latestVersionCode: 4,
        issuedAt: freshIssuedAt,
        forceUpdate: true,
      );
      final svc = serviceWith(manifest, store: store);
      final info = await svc.checkForUpdate(
        packageName: testPackageName,
        versionCode: 3,
      );
      expect(info, isNotNull);
      expect(info!.forceUpdate, isTrue);

      expect(
        await svc.consentAndLaunchUpdate(info, packageName: testPackageName),
        isTrue,
      );
      final replay = await serviceWith(
        manifest,
        store: store,
      ).checkForUpdate(packageName: testPackageName, versionCode: 3);
      expect(replay, isNull);
    });

    test('returns the launcher result even when launch throws', () async {
      final svc = serviceWith(
        await _signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
        ),
        launcher: (uri) async => throw Exception('no handler'),
      );
      expect(
        await svc.openUpdateLink('https://updates.example.com/app.apk'),
        isFalse,
      );
    });
  });

  group('UpdateService.verifyApkSha256', () {
    final apkBytes = utf8.encode('this is the apk content');
    final correctHash = sha256.convert(apkBytes).toString();

    test('accepts bytes whose sha256 matches the manifest hash', () {
      expect(UpdateService.verifyApkSha256(apkBytes, correctHash), isTrue);
    });

    test('rejects tampered bytes', () {
      final tampered = [...apkBytes, 0];
      expect(UpdateService.verifyApkSha256(tampered, correctHash), isFalse);
    });

    test('rejects missing / malformed manifest hash', () {
      expect(UpdateService.verifyApkSha256(apkBytes, null), isFalse);
      expect(UpdateService.verifyApkSha256(apkBytes, 'abcd'), isFalse);
    });

    test('accepts uppercase manifest hash (normalized to lowercase)', () {
      expect(
        UpdateService.verifyApkSha256(apkBytes, correctHash.toUpperCase()),
        isTrue,
      );
    });
  });

  group('UpdateService.downloadAndVerifyApk (controlled download)', () {
    test('returns the verified local path for matching bytes', () async {
      final manifest = await _signedManifest(
        apkSha256: apkSha256,
        latestVersionCode: 4,
        issuedAt: freshIssuedAt,
      );
      final svc = serviceWith(manifest);
      final path = await svc.downloadAndVerifyApk(
        url: 'https://updates.example.com/app-v1.2.3.apk',
        expectedSha256: apkSha256,
      );
      expect(path, isNotEmpty);
      expect(File(path).existsSync(), isTrue);
    });

    test('fails closed on hash mismatch (tampered download bytes)', () async {
      final manifest = await _signedManifest(
        apkSha256: apkSha256,
        latestVersionCode: 4,
        issuedAt: freshIssuedAt,
      );
      final svc = serviceWith(
        manifest,
        downloadBytes: Uint8List.fromList([1, 2, 3, 4, 5]),
      );
      await expectLater(
        svc.downloadAndVerifyApk(
          url: 'https://updates.example.com/app-v1.2.3.apk',
          expectedSha256: apkSha256,
        ),
        throwsA(isA<UpdateIntegrityException>()),
      );
    });

    test('fails closed when the manifest has no usable sha256', () async {
      final svc = serviceWith({});
      await expectLater(
        svc.downloadAndVerifyApk(
          url: 'https://updates.example.com/app-v1.2.3.apk',
          expectedSha256: null,
        ),
        throwsA(isA<UpdateIntegrityException>()),
      );
    });

    test(
      'consentAndLaunchUpdate fails closed (no consent) on hash mismatch',
      () async {
        final store = InMemoryManifestStateStore();
        final manifest = await _signedManifest(
          apkSha256: apkSha256,
          latestVersionCode: 4,
          issuedAt: freshIssuedAt,
          forceUpdate: true,
        );
        final info = await serviceWith(
          manifest,
        ).checkForUpdate(packageName: testPackageName, versionCode: 3);
        expect(info, isNotNull);
        final svc = serviceWith(
          manifest,
          store: store,
          downloadBytes: Uint8List.fromList([9, 9, 9]),
        );
        expect(
          await svc.consentAndLaunchUpdate(info!, packageName: testPackageName),
          isFalse,
        );
        final again = await serviceWith(
          manifest,
          store: store,
        ).checkForUpdate(packageName: testPackageName, versionCode: 3);
        expect(
          again,
          isNotNull,
          reason: 'hash mismatch must not suppress the update',
        );
      },
    );
  });
}
