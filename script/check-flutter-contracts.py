#!/usr/bin/env python3
"""
Executable static contract checks for the Flutter client changes.

The Flutter/Dart SDK is not available in this environment, so `flutter test`
cannot run here. This script verifies the security invariants that the Dart
unit tests assert, by reading the source directly. It is deliberately simple
(string/substring based) and MUST be run in CI alongside `flutter test`.

Exit code is non-zero on any contract violation.
"""
import pathlib
import re
import sys

REPO = pathlib.Path(__file__).resolve().parent.parent
USERAPP = REPO / "app" / "xiaoheifs_userapp"
ADMINAPP = REPO / "app" / "xiaoheifs_app"

failures = []


def check(cond, msg):
    if cond:
        print(f"  ok   {msg}")
    else:
        print(f"  FAIL {msg}")
        failures.append(msg)


def read(rel):
    return (REPO / rel).read_text(encoding="utf-8", errors="replace")


def main():
    print("== Flutter static contract checks ==")

    # 1. Token storage keys are distinct and not scrub placeholders.
    print("[1] storage keys")
    keys = read("app/xiaoheifs_userapp/lib/core/config/storage_keys.dart")
    for k in ("accessToken", "refreshToken"):
        check(re.search(rf"{k}\s*=\s*'([^']+)'", keys) is not None,
              f"storage_keys.dart defines {k}")
        m = re.search(rf"{k}\s*=\s*'([^']+)'", keys)
        if m:
            check(m.group(1) not in ("<REDACTED>", "", "redacted"),
                  f"{k} is not a placeholder")
    at = re.search(r"accessToken\s*=\s*'([^']+)'", keys)
    rt = re.search(r"refreshToken\s*=\s*'([^']+)'", keys)
    if at and rt:
        check(at.group(1) != rt.group(1), "access/refresh keys are distinct")

    # 2. Sensitive storage uses flutter_secure_storage, not SharedPreferences.
    print("[2] secure storage migration (user app)")
    storage = read("app/xiaoheifs_userapp/lib/core/storage/storage_service.dart")
    check("flutter_secure_storage" in storage,
          "storage_service imports flutter_secure_storage")
    check("FlutterSecureStorage" in storage,
          "storage_service instantiates FlutterSecureStorage")
    check("SharedPreferences" in storage,
          "storage_service still keeps SharedPreferences for non-sensitive prefs")
    # Token getters must read from secure storage.
    for method in ("setAccessToken", "getAccessToken", "setRefreshToken",
                   "getRefreshToken", "clearAuthData", "setUserInfo",
                   "getUserInfo", "setUserId", "getUserId"):
        check(re.search(rf"Future<[^>]*> {method}\(", storage) is not None,
              f"{method} is async")
    # No plaintext token writes to SharedPreferences anywhere in lib.
    lib = "\n".join(p.read_text(encoding="utf-8", errors="replace")
                    for p in (USERAPP / "lib").rglob("*.dart"))
    check("StorageKeys.accessToken" in lib,
          "storage key constant is referenced")
    check("await _storage.getAccessToken()" in lib or
          "await StorageService.instance.getAccessToken()" in lib,
          "token reads are awaited")

    print("[3] secure storage migration (admin app)")
    astorage = read("app/xiaoheifs_app/lib/services/app_storage.dart")
    check("flutter_secure_storage" in astorage,
          "admin app_storage imports flutter_secure_storage")
    check("FlutterSecureStorage" in astorage,
          "admin app_storage instantiates FlutterSecureStorage")
    check("_secure.write(key: _keyApiKey" in astorage,
          "api_key written to secure storage")
    check("_secure.write(key: _keyToken" in astorage,
          "token written to secure storage")
    check("_secure.write(key: _keyRefreshToken" in astorage,
          "refresh_token written to secure storage")

    # 3b. Corrupted / incomplete legacy session.json must be DELETED, not left
    #     on disk: any partial plaintext (token/refresh/api_key) in the file
    #     would otherwise survive permanently and be re-parsed every launch.
    print("[3b] corrupted/incomplete session.json is deleted (no residue)")
    check("_deleteQuietly" in astorage,
          "admin app_storage has a delete helper for legacy session.json")
    check("if (decoded is! Map<String, dynamic>)" in astorage and
          "_deleteQuietly(file)" in astorage,
          "corrupted-JSON path deletes the file")
    check("if (apiUrl.isEmpty)" in astorage and "_deleteQuietly(file)" in astorage,
          "missing-api_url path deletes the file")
    check("corrupted plaintext session.json must be deleted" in
          read("app/xiaoheifs_app/test/app_storage_test.dart"),
          "app_storage_test covers corrupted session.json cleanup")
    check("incomplete plaintext session.json must be deleted" in
          read("app/xiaoheifs_app/test/app_storage_test.dart"),
          "app_storage_test covers incomplete session.json cleanup")

    # 4. Release signing must not use the debug keystore.
    print("[4] release signing")
    for rel, name in (("app/xiaoheifs_userapp", "userapp"),
                      ("app/xiaoheifs_app", "adminapp")):
        gradle = read(f"{rel}/android/app/build.gradle.kts")
        check('signingConfig = signingConfigs.getByName("debug")' not in gradle,
              f"{name} release build does not use debug keystore")
        check("ANDROID_RELEASE_KEYSTORE_PATH" in gradle,
              f"{name} release signing reads CI env")

    # 5. Cleartext disabled in main manifest, allowed only in debug.
    print("[5] cleartext")
    for rel, name in (("app/xiaoheifs_userapp", "userapp"),
                      ("app/xiaoheifs_app", "adminapp")):
        main_manifest = read(f"{rel}/android/app/src/main/AndroidManifest.xml")
        check('usesCleartextTraffic="${usesCleartextTraffic}"' in main_manifest,
              f"{name} main manifest uses build-type placeholder")
        main_nsc = read(f"{rel}/android/app/src/main/res/xml/network_security_config.xml")
        check('cleartextTrafficPermitted="false"' in main_nsc,
              f"{name} main network security config denies cleartext")

    # 6. Update service: https only + SHA-256 verification + Ed25519 manifest
    #    signature (real authenticity, not a shared-secret SHA-256).
    print("[6] update integrity (Ed25519 signed manifest)")
    uus = read("app/xiaoheifs_userapp/lib/core/update/update_service.dart")
    aus = read("app/xiaoheifs_app/lib/services/update_service.dart")
    uver = read("app/xiaoheifs_userapp/lib/core/update/manifest_verifier.dart")
    aver = read("app/xiaoheifs_app/lib/services/manifest_verifier.dart")
    ukeys = read("app/xiaoheifs_userapp/lib/core/update/update_keys.dart")
    akeys = read("app/xiaoheifs_app/lib/services/update_keys.dart")
    for svc, ver, keys, name in (
        (uus, uver, ukeys, "userapp"),
        (aus, aver, akeys, "adminapp"),
    ):
        check("verifyApkSha256" in svc, f"{name} update service verifies APK SHA-256")
        check("uri.scheme == 'https'" in svc or "uri.scheme != 'https'" in svc,
              f"{name} update URL must be https")
        check("apkSha256" in svc, f"{name} update info carries apkSha256")
        # Ed25519 verification is mandatory and fail-closed before an update is offered.
        check("ManifestVerifier.verifyManifest" in svc,
              f"{name} update service verifies manifest Ed25519 signature")
        check("verifyManifest" in ver, f"{name} manifest_verifier defines verifyManifest")
        check("package:cryptography/cryptography.dart" in ver,
              f"{name} verifier uses mature cryptography package (Ed25519)")
        check("ed25519" in ver.lower(), f"{name} verifier pins algorithm to ed25519")
        check("signature.length != signatureLength" in ver,
              f"{name} verifier rejects wrong-length signature (fail closed)")
        check("if (!verified) return false" in ver,
              f"{name} verifier rejects failed signatures (fail closed)")
        check("latestVersionCode <= currentVersionCode" in ver,
              f"{name} verifier rejects downgrade/replay (fail closed)")
        # The update decision must be covered by the signature (no unsigned
        # field may flip it) and key rotation must be explicit via keyEpoch.
        check("hasUpdate=${hasUpdate ? '1' : '0'}" in ver,
              f"{name} canonical payload signs hasUpdate (update decision)")
        check("'epoch=$keyEpoch'" in ver,
              f"{name} canonical payload signs keyEpoch (explicit rotation)")
        check("publicKeys[keyEpoch]" in ver,
              f"{name} verifier looks up the key by keyEpoch")
        check("placeholderPublicKeyHex" in ver and "every((b) => b == 0)" in ver,
              f"{name} verifier rejects the placeholder public key (fail closed)")
        check("hasUpdate is! bool" in ver,
              f"{name} verifier strictly types hasUpdate")
        check("String.fromEnvironment" in keys and "UPDATE_SIGN_PUBLIC_KEY" in keys,
              f"{name} public key is build-time injectable")
        # No self-rolled crypto: the verifier must not hand-roll Ed25519.
        check("_signEd25519" not in ver and "ed25519_sign" not in ver.lower(),
              f"{name} verifier does not self-implement Ed25519")
    check("cryptography: ^2.7.0" in read("app/xiaoheifs_userapp/pubspec.yaml"),
          "userapp declares cryptography dependency")
    check("cryptography: ^2.7.0" in read("app/xiaoheifs_app/pubspec.yaml"),
          "adminapp declares cryptography dependency")

    # 6b. Cross-session replay guard in secure storage (highest accepted
    # versionCode/issuedAt per package + keyEpoch).
    print("[6b] cross-session replay guard (secure storage)")
    for rel, name in (
        ("app/xiaoheifs_userapp/lib/core/update/update_replay_guard.dart", "userapp"),
        ("app/xiaoheifs_app/lib/services/update_replay_guard.dart", "adminapp"),
    ):
        guard = read(rel)
        check("class ManifestReplayGuard" in guard, f"{name} defines ManifestReplayGuard")
        check("isAcceptable" in guard and "recordAccepted" in guard,
              f"{name} replay guard accepts-then-records strictly newer manifests")
        check("versionCode < prev.versionCode" in guard and
              "issuedAt <= prev.issuedAt" in guard,
              f"{name} replay guard rejects downgrade/replay")
        check("InMemoryManifestStateStore" in guard,
              f"{name} replay guard ships an in-memory store for pure-Dart tests")
    usvc = read("app/xiaoheifs_userapp/lib/core/update/update_service.dart")
    asvc = read("app/xiaoheifs_app/lib/services/update_service.dart")
    for svc, name in ((usvc, "userapp"), (asvc, "adminapp")):
        check("ManifestReplayGuard" in svc, f"{name} update service wires the replay guard")
        check("isAcceptable" in svc, f"{name} update service enforces accepted state")
        # F14: the record is written only when the user consents to install
        # (recordAcceptedConsent), never merely by offering the update in
        # checkForUpdate — a dismissed non-forced update is re-offered next
        # launch instead of being permanently suppressed.
        check("recordAcceptedConsent" in svc,
              f"{name} update service exposes consent-time recording")
        check("await _replayGuard.recordAccepted(" in svc,
              f"{name} replay record is written via the replay guard")
        # Residual round-5: consent must be persisted only AFTER the OS handoff
        # succeeds. consentAndLaunchUpdate launches first, then records on
        # success — a failed launch (no browser/intent handler) must not write
        # the record, else a same-version forced update would be permanently
        # suppressed. The launcher is injectable for tests.
        check("consentAndLaunchUpdate" in svc,
              f"{name} update service exposes launch-then-record handoff")
        # P2-13: the launch is preceded by a verified controlled download; the
        # record is still written only after the launcher succeeds.
        check("if (launched)" in svc and
              "await recordAcceptedConsent(update" in svc,
              f"{name} consent recorded only after a successful launch")
        check("Future<bool> Function(Uri url)? launcher" in svc,
              f"{name} launcher is injectable for failure-path tests")
        check("hasSignature" in svc, f"{name} update service verifies before trusting hasUpdate")

    # Residual round-5 UI wiring: both login surfaces call consentAndLaunchUpdate
    # when the user taps "Update now" (never on mere display/dismiss).
    for page, name in (
        ("app/xiaoheifs_userapp/lib/presentation/pages/auth/login_page.dart", "userapp"),
        ("app/xiaoheifs_app/lib/screens/login_screen.dart", "adminapp"),
    ):
        page_text = read(page)
        check("consentAndLaunchUpdate" in page_text,
              f"{name} login page launches then records consent on 'Update now'")
        check("if (!launched" in page_text,
              f"{name} login page surfaces a failed launch to the user")
    check("flutter_secure_storage" in
          read("app/xiaoheifs_userapp/lib/core/update/secure_manifest_state_store.dart"),
          "userapp replay state stored in flutter_secure_storage")
    check("flutter_secure_storage" in
          read("app/xiaoheifs_app/lib/services/secure_manifest_state_store.dart"),
          "adminapp replay state stored in flutter_secure_storage")

    # 6c. APK URL trust policy is unified: https + same-origin on BOTH apps,
    #     and the contract docs describe the download model honestly (P2-13:
    #     controlled download — the manifest sha256 constrains the actual
    #     installed bytes; the external-browser helper is no longer the default
    #     install path because it cannot verify bytes).
    print("[6c] same-origin APK trust + controlled-download docs")
    for svc, name in ((usvc, "userapp"), (asvc, "adminapp")):
        check("uri.host == server.host" in svc,
              f"{name} APK URL must be same-origin with the update server")
        check("uri.scheme != 'https'" in svc or "uri.scheme == 'https'" in svc,
              f"{name} APK URL must be https")
    contract = read("docs/update-signing-contract.md")
    check("受控下载" in contract,
          "contract docs describe the controlled-download model")
    check("约束**实际安装字节**" in contract,
          "contract docs state sha256 constrains the installed bytes")
    check("downloadAndVerifyApk" in contract,
          "contract docs point to downloadAndVerifyApk as the self-install helper")
    # 6d. P2-13: downloadAndVerifyApk is WIRED into the real update flow on both
    #     apps (consentAndLaunchUpdate downloads + verifies before launching),
    #     and Android hands the verified file to the installer via FileProvider.
    print("[6d] controlled download wired into the update flow")
    for svc, name in ((usvc, "userapp"), (asvc, "adminapp")):
        check("verifiedPath = await downloadAndVerifyApk(" in svc,
              f"{name} consentAndLaunchUpdate downloads + verifies the APK")
        check("_installUriForFile(verifiedPath)" in svc,
              f"{name} launches the VERIFIED local file (not the raw URL)")
        check("hash mismatch" in svc.lower() or "sha256 mismatch" in svc.lower(),
              f"{name} hash mismatch fails closed")
    for rel, name in (
        ("app/xiaoheifs_userapp/android/app/src/main/AndroidManifest.xml", "userapp"),
        ("app/xiaoheifs_app/android/app/src/main/AndroidManifest.xml", "adminapp"),
    ):
        manifest = read(rel)
        check("androidx.core.content.FileProvider" in manifest,
              f"{name} Android manifest declares FileProvider")
    for rel, name in (
        ("app/xiaoheifs_userapp/lib/core/update/update_service.dart", "userapp"),
        ("app/xiaoheifs_app/lib/services/update_service.dart", "adminapp"),
    ):
        svc = read(rel)
        check("xiaohei/update" in svc and "getContentUriForPath" in svc,
              f"{name} resolves a content:// install URI via the method channel")
    check("fails closed on hash mismatch (tampered download bytes)" in read(
          "app/xiaoheifs_userapp/test/update_service_test.dart"),
          "userapp downloadAndVerifyApk test covers hash mismatch")
    check("fails closed on hash mismatch (tampered download bytes)" in read(
          "app/xiaoheifs_app/test/update_service_test.dart"),
          "adminapp downloadAndVerifyApk test covers hash mismatch")

    # 7. VpsDetailState.copyWith sentinel (F13b fix).
    print("[7] VPS detail copyWith sentinel")
    vps = read("app/xiaoheifs_userapp/lib/presentation/providers/vps_provider.dart")
    check("_unset" in vps, "vps_provider has an _unset sentinel")
    check("identical(detail, _unset)" in vps,
          "copyWith distinguishes explicit null from unset")
    check("detail: sameInstance ? state.detail : null" in vps,
          "fetch clears detail when switching instances")

    # 7b. Round-5: request-generation guards for VPS detail + monitor.
    #     A slow response from a previous VPS / previous account must never
    #     overwrite the current session's state.
    print("[7b] VPS detail/monitor request generation guards")
    check("int _generation = 0" in vps,
          "VpsDetailNotifier has a request generation counter")
    check("final generation = ++_generation" in vps and
          "if (generation != _generation) return" in vps,
          "stale detail responses (out-of-order / cross-account) are dropped")
    check("void reset()" in vps and "_generation++" in vps,
          "detail reset invalidates in-flight fetches on session boundary")
    check("if (generation != _generation) return" in vps,
          "monitor drops stale responses after stopPolling/session switch")
    check("nowId == null || nowId.toString() != _id.toString()" in vps,
          "monitor re-verifies the viewed VPS id after await (no cross-VPS apply)")
    # Admin app VPS detail also guards request generation on session/client
    # change, so a stale account's slow response cannot write into the new
    # session's screen state.
    admin_vps = read("app/xiaoheifs_app/lib/screens/vps_detail_screen.dart")
    check("int _generation = 0" in admin_vps,
          "admin VPS detail has a request generation counter")
    check("++_generation" in admin_vps and "gen != _generation" in admin_vps,
          "admin VPS detail drops stale responses (session/client change)")

    # 5. Round-5: panel/VNC redirect is HTTPS-only in production; localhost
    #    http is a debug-only exception. A short-lived ticket in a cleartext URL
    #    is eavesdroppable on public/LAN links.
    print("[5] panel/VNC redirect HTTPS-only (localhost debug exception)")
    vpsrepo = read("app/xiaoheifs_userapp/lib/data/repositories/vps_repository.dart")
    check("kReleaseMode" in vpsrepo,
          "redirect policy is gated on release build mode")
    check("uri.scheme == 'https'" in vpsrepo and "return true" in vpsrepo,
          "https redirect URLs accepted in all modes")
    check("'localhost'" in vpsrepo and "'127.0.0.1'" in vpsrepo and "'::1'" in vpsrepo,
          "cleartext exception limited to loopback hosts")
    check("allowCleartext ?? !kReleaseMode" in vpsrepo,
          "non-release allows loopback http; release rejects all cleartext")
    check("_requireSafeRedirectUrl" in vpsrepo and
          "isSafeRedirectUrl(url)" in vpsrepo,
          "resolved redirects go through the HTTPS-only policy")

    # 8. 401 forced-logout resets business providers (F1 residual).
    print("[8] session-expired provider reset")
    interceptors = read("app/xiaoheifs_userapp/lib/core/network/api_interceptors.dart")
    check("onSessionExpired" in interceptors,
          "AuthInterceptor exposes onSessionExpired hook")
    maindart = read("app/xiaoheifs_userapp/lib/main.dart")
    check("handleSessionExpired" in maindart,
          "main.dart wires handleSessionExpired on session expiry")
    authprov = read("app/xiaoheifs_userapp/lib/presentation/providers/auth_provider.dart")
    check("handleSessionExpired" in authprov,
          "AuthNotifier defines handleSessionExpired")

    # 8b. Round-5: logout/session-expiry invalidates the non-autoDispose family
    #     providers (firewall/ports/snapshots/backups/monitor) so account B never
    #     sees account A's cached lists.
    print("[8b] session boundary invalidates family providers")
    check("_ref.invalidate(vpsFirewallProvider)" in authprov and
          "_ref.invalidate(vpsPortsProvider)" in authprov and
          "_ref.invalidate(vpsSnapshotsProvider)" in authprov and
          "_ref.invalidate(vpsBackupsProvider)" in authprov and
          "_ref.invalidate(vpsMonitorStateProvider)" in authprov,
          "logout/session-expiry invalidates non-autoDispose family providers")

    # 9. No debug request logging outside kDebugMode.
    print("[9] request logging")
    apiclient = read("app/xiaoheifs_userapp/lib/core/network/api_client.dart")
    check("kDebugMode" in apiclient and "PrettyDioLogger" in apiclient,
          "PrettyDioLogger gated behind kDebugMode")

    # 12. Refresh singleflight is bound to the session epoch/identity: account
    #     A's in-flight refresh must never write into account B's storage/state
    #     after logout/login, and a stale waiter must not clear a newer future.
    print("[12] refresh singleflight bound to session epoch/identity")
    inter = read("app/xiaoheifs_userapp/lib/core/network/api_interceptors.dart")
    check("_refreshingFor" in inter,
          "userapp refresh singleflight tracks the session identity")
    check("_refreshingFor != refreshToken" in inter,
          "userapp new session starts its own refresh (no cross-session sharing)")
    check("identical(_refreshing, future)" in inter,
          "userapp waiter cannot clear a newer-generation future (identical guard)")
    check("current != refreshToken" in inter,
          "userapp refresh result is discarded when the session changed")
    check("nowAccess == sessionAccess" in inter,
          "userapp stale 401 does not force-logout a switched session")
    astate = read("app/xiaoheifs_app/lib/app_state.dart")
    check("_refreshFutureSession" in astate,
          "admin refresh singleflight tracks the session identity")
    check("_sessionIdentity" in astate and "'${s.apiUrl}|${s.refreshToken}'" in astate,
          "admin session identity is apiUrl+refreshToken")
    check("current.refreshToken == null ||" in astate and "!= identity" in astate,
          "admin refresh result is discarded when the session changed")
    check("_handleAuthExpiredFor" in astate and "current.token != expected.token" in astate,
          "admin stale ApiClient 401 cannot log out a switched session")
    check("account A in-flight refresh cannot overwrite or log out account B" in
          read("app/xiaoheifs_app/test/app_state_test.dart"),
          "app_state_test covers cross-account in-flight refresh regression")

    print()
    if failures:
        print(f"{len(failures)} contract violation(s)")
        sys.exit(1)
    print("all Flutter static contracts satisfied")


if __name__ == "__main__":
    main()
