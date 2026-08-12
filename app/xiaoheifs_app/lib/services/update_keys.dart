/// Update-manifest verification public keys (Ed25519, hex, 32 bytes), keyed by
/// keyEpoch.
///
/// The release build MUST inject the production public key for epoch 1 via
/// `--dart-define=UPDATE_SIGN_PUBLIC_KEY=<hex>`; without it the all-zero
/// placeholder is used and the verifier fails closed (see
/// `manifest_verifier.dart`'s `placeholderPublicKeyHex`). A build without a
/// real key therefore never accepts an update — "no key configured" is never a
/// pass.
///
/// Key rotation: add the new epoch's public key to this map (or via
/// `--dart-define`) while keeping the old epoch's key in the same build for an
/// explicit dual-key transition; the publisher signs with the new epoch and the
/// client picks the key by the manifest's `keyEpoch`. The private key is held
/// by the release maintainers only — never committed or shipped with the app
/// (see `docs/update-signing-contract.md`).
const String _injectedPublicKey = String.fromEnvironment(
  'UPDATE_SIGN_PUBLIC_KEY',
);

/// keyEpoch -> Ed25519 public key hex. The default epoch-1 entry is the
/// fail-closed placeholder.
final Map<int, String> updateSignPublicKeys = <int, String>{
  1: _injectedPublicKey.isEmpty
      ? '0000000000000000000000000000000000000000000000000000000000000000'
      : _injectedPublicKey,
};
