import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:crypto/crypto.dart';
import 'package:cryptography/cryptography.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opensource_userapp/core/config/update_config.dart';
import 'package:opensource_userapp/core/update/manifest_verifier.dart';
import 'package:opensource_userapp/core/update/update_replay_guard.dart';
import 'package:opensource_userapp/core/update/update_service.dart';

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

/// Stub Dio adapter: serves the canned JSON manifest for the update-check path,
/// and the real APK bytes for the .apk download path (so downloadAndVerifyApk
/// can verify against the manifest's sha256).
class _StubAdapter implements HttpClientAdapter {
  _StubAdapter(this.payload, {this.apkBytes});
  final Map<String, dynamic> payload;
  final Uint8List? apkBytes;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    final path = options.path;
    if (apkBytes != null && path.toLowerCase().endsWith('.apk')) {
      return ResponseBody.fromBytes(apkBytes!, 200);
    }
    return ResponseBody.fromString(
      jsonEncode(payload),
      200,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }
}

Future<Map<String, dynamic>> _signedManifest({
  required String apkSha256,
  required int latestVersionCode,
  required int issuedAt,
  bool hasUpdate = true,
  int keyEpoch = 1,
  String latestVersion = '1.2.3',
  bool forceUpdate = false,
  String apkUrl = 'https://pkg.example.com/app-v1.2.3.apk',
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
    UpdateConfig.serverBaseUrl = 'https://pkg.example.com';
  });

  UpdateService serviceWith(
    Map<String, dynamic> payload, {
    ManifestStateStore? store,
    Future<bool> Function(Uri url)? launcher,
    Uint8List? downloadBytes,
  }) {
    final dio = Dio()
      ..httpClientAdapter = _StubAdapter(
        payload,
        apkBytes: downloadBytes ?? apkBytes,
      );
    return UpdateService(
      dio: dio,
      publicKeys: pubKeys,
      clock: () => now,
      stateStore: store ?? InMemoryManifestStateStore(),
      launcher: launcher ?? (uri) async => true,
    );
  }

  group('UpdateService.isSafeDownloadUrl', () {
    test('accepts https download URLs only (release disables cleartext)', () {
      expect(
        UpdateService.isSafeDownloadUrl('https://pkg.example.com/app.apk'),
        isTrue,
      );
      expect(
        UpdateService.isSafeDownloadUrl('http://pkg.example.com/app.apk'),
        isFalse,
      );
    });

    test('rejects an https URL on a different host than the update server '
        '(same-origin trust, matches adminapp)', () {
      // Same origin as UPDATE_SERVER_URL (pkg.example.com) -> accepted.
      expect(
        UpdateService.isSafeDownloadUrl('https://pkg.example.com/app.apk'),
        isTrue,
      );
      // Signed manifest may point anywhere, but https + cross-host is refused:
      // the APK must live on the configured update server.
      expect(
        UpdateService.isSafeDownloadUrl('https://cdn.example.com/app.apk'),
        isFalse,
      );
      expect(
        UpdateService.isSafeDownloadUrl(
          'https://pkg.example.com.evil.io/x.apk',
        ),
        isFalse,
      );
    });

    test('rejects non-https schemes that could launch arbitrary intents', () {
      expect(
        UpdateService.isSafeDownloadUrl('intent://evil#Intent;end'),
        isFalse,
      );
      expect(
        UpdateService.isSafeDownloadUrl('file:///sdcard/evil.apk'),
        isFalse,
      );
      expect(UpdateService.isSafeDownloadUrl('javascript:alert(1)'), isFalse);
      expect(
        UpdateService.isSafeDownloadUrl('content://provider/evil.apk'),
        isFalse,
      );
      expect(
        UpdateService.isSafeDownloadUrl('ftp://pkg.example.com/app.apk'),
        isFalse,
      );
    });

    test('rejects malformed or empty URLs', () {
      expect(UpdateService.isSafeDownloadUrl(null), isFalse);
      expect(UpdateService.isSafeDownloadUrl(''), isFalse);
      expect(UpdateService.isSafeDownloadUrl('not a url'), isFalse);
    });
  });

  group('UpdateService.checkForUpdate (signed manifest)', () {
    test('accepts a correctly signed update', () async {
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
      expect(info!.apkUrl, 'https://pkg.example.com/app-v1.2.3.apk');
      expect(info.apkSha256, apkSha256);
    });

    test('rejects an unsigned manifest (fail closed)', () async {
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

    test('rejects a manifest with a tampered artifact hash', () async {
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

    test('rejects a manifest with a tampered URL', () async {
      final payload = await _signedManifest(
        apkSha256: apkSha256,
        latestVersionCode: 4,
        issuedAt: freshIssuedAt,
      );
      payload['apkUrl'] = 'https://evil.example.com/trojan.apk';
      final svc = serviceWith(payload);
      final info = await svc.checkForUpdate(
        packageName: testPackageName,
        versionCode: 3,
      );
      expect(info, isNull);
    });

    test('rejects a manifest signed with the wrong key', () async {
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
        apkUrl: 'https://pkg.example.com/app-v1.2.3.apk',
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
        'apkUrl': 'https://pkg.example.com/app-v1.2.3.apk',
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
      // A user who taps "稍后" must see the same update again on the next
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

        // The user taps "立即更新": only now is the record written.
        await svc.recordAcceptedConsent(info!, packageName: testPackageName);

        // Replaying the same signed manifest later (e.g. an attacker holds a
        // copy) must be rejected: the client has accepted this version.
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
        // The signature covered hasUpdate=true; flipping it invalidates the
        // signature, so the client must not treat it as a valid no-update.
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
          issuedAt: freshIssuedAt - 8 * 24 * 3600,
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
          latestVersionCode: 3,
          issuedAt: freshIssuedAt,
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
          apkUrl: 'http://pkg.example.com/app.apk',
        ),
      );
      final info = await svc.checkForUpdate(
        packageName: testPackageName,
        versionCode: 3,
      );
      expect(info, isNull);
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

    test(
      'returns false when the launcher throws (no intent handler)',
      () async {
        final svc = serviceWith(
          await _signedManifest(
            apkSha256: apkSha256,
            latestVersionCode: 4,
            issuedAt: freshIssuedAt,
          ),
          launcher: (uri) async => throw Exception('no handler'),
        );
        expect(
          await svc.openUpdateLink('https://pkg.example.com/app.apk'),
          isFalse,
        );
      },
    );
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
      expect(
        UpdateService.verifyApkSha256(
          apkBytes,
          'ZZ${correctHash.substring(2)}',
        ),
        isFalse,
      );
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
        url: 'https://pkg.example.com/app-v1.2.3.apk',
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
      // Serve bytes that do NOT match the manifest sha256.
      final svc = serviceWith(
        manifest,
        downloadBytes: Uint8List.fromList([1, 2, 3, 4, 5]),
      );
      await expectLater(
        svc.downloadAndVerifyApk(
          url: 'https://pkg.example.com/app-v1.2.3.apk',
          expectedSha256: apkSha256,
        ),
        throwsA(isA<UpdateIntegrityException>()),
      );
    });

    test('fails closed when the manifest has no usable sha256', () async {
      final svc = serviceWith({});
      await expectLater(
        svc.downloadAndVerifyApk(
          url: 'https://pkg.example.com/app-v1.2.3.apk',
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
        // Tampered download bytes => consentAndLaunchUpdate must fail closed and
        // NOT record consent, so the same-version forced update is offered again.
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
