/// 更新清单验签公钥表（Ed25519，hex，32 字节），按 keyEpoch 索引。
///
/// 构建期可用 `--dart-define=UPDATE_SIGN_PUBLIC_KEY=<hex>` 注入 **epoch 1** 的
/// 生产公钥；未注入时使用全零占位公钥。验签器对占位公钥一律 fail-closed
/// （见 `manifest_verifier.dart` 的 `placeholderPublicKeyHex`），因此不带
/// 生产公钥的发布构建不会放行任何更新——这是刻意的：发布构建**必须**注入
/// 真实公钥，缺省「未配置 = 放行」是禁止的。
///
/// 密钥轮换：新增 epoch 的公钥在源码中追加（例如 keyEpoch 2 的 `--dart-define`
/// 或后续编译期常量），旧 epoch 在新构建中保留，实现显式双 key 过渡；发布端
/// 用新 epoch 签名，客户端按清单中的 `keyEpoch` 选择对应公钥。私钥由发布
/// 维护者持有（绝不进入仓库或客户端），轮换流程见
/// `docs/update-signing-contract.md`。
const String _injectedPublicKey = String.fromEnvironment(
  'UPDATE_SIGN_PUBLIC_KEY',
);

/// keyEpoch -> Ed25519 公钥 hex。默认 epoch 1 为占位公钥（fail-closed）。
final Map<int, String> updateSignPublicKeys = <int, String>{
  1: _injectedPublicKey.isEmpty
      ? '0000000000000000000000000000000000000000000000000000000000000000'
      : _injectedPublicKey,
};
