/// 应用自更新配置
class UpdateConfig {
  const UpdateConfig._();

  /// 更新服务器地址，构建期通过 `--dart-define=UPDATE_SERVER_URL=...` 注入
  /// （避免发布版硬编码占位域名导致更新检查永远失败）；运行期可覆盖（测试）。
  static String serverBaseUrl = String.fromEnvironment(
    'UPDATE_SERVER_URL',
    defaultValue: 'https://pkg.example.com',
  );

  /// 更新检查接口路径
  static const String checkPath = '/api/v1/update/check';
}
