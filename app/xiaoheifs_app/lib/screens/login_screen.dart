import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';
import '../services/admin_auth.dart';
import '../services/api_client.dart';
import '../services/app_storage.dart';
import '../services/update_service.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen>
    with SingleTickerProviderStateMixin {
  final _apiKeyFormKey = GlobalKey<FormState>();
  final _passwordFormKey = GlobalKey<FormState>();

  final _apiUrlController = TextEditingController(text: 'https://api.example.com');
  final _apiKeyController = TextEditingController();
  final _usernameController = TextEditingController(text: 'Admin');

  final _loginUserController = TextEditingController(text: 'admin');
  final _loginPasswordController = TextEditingController();

  late final TabController _tabController;
  bool _saving = false;
  String? _error;
  static bool _hasCheckedUpdateInThisLaunch = false;
  final UpdateService _updateService = UpdateService();

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 2, vsync: this);
    _loadLastSession();
    _checkAppUpdateIfNeeded();
  }

  @override
  void dispose() {
    _apiUrlController.dispose();
    _apiKeyController.dispose();
    _usernameController.dispose();
    _loginUserController.dispose();
    _loginPasswordController.dispose();
    _tabController.dispose();
    super.dispose();
  }

  Future<void> _loadLastSession() async {
    final storage = AppStorage();
    final session = await storage.loadSession();
    if (session == null) return;
    _apiUrlController.text = session.apiUrl;
    _usernameController.text = session.username;
    if (session.apiKey != null && session.apiKey!.isNotEmpty) {
      _apiKeyController.text = session.apiKey!;
    }
  }

  Future<void> _checkAppUpdateIfNeeded() async {
    if (_hasCheckedUpdateInThisLaunch) return;
    _hasCheckedUpdateInThisLaunch = true;
    if (kIsWeb || defaultTargetPlatform != TargetPlatform.android) return;

    try {
      final info = await PackageInfo.fromPlatform();
      final buildNumber = int.tryParse(info.buildNumber) ?? 0;
      final update = await _updateService.checkForUpdate(
        packageName: info.packageName,
        versionCode: buildNumber,
      );
      if (!mounted || update == null || !update.hasUpdate) return;
      await _showUpdateDialog(update);
    } catch (_) {
      // Keep login flow unaffected if update check fails.
    }
  }

  Future<void> _showUpdateDialog(AppUpdateInfo update) async {
    await showDialog<void>(
      context: context,
      barrierDismissible: !update.forceUpdate,
      builder: (context) {
        return PopScope(
          canPop: !update.forceUpdate,
          child: AlertDialog(
            title: Text('发现新版本 V${update.latestVersion}'),
            content: SingleChildScrollView(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text('版本号: ${update.latestVersionCode}'),
                  const SizedBox(height: 10),
                  const Text('更新说明:'),
                  const SizedBox(height: 6),
                  Text(update.changelog.isEmpty ? '暂无更新说明' : update.changelog),
                ],
              ),
            ),
            actions: [
              if (!update.forceUpdate)
                TextButton(
                  onPressed: () => Navigator.of(context).pop(),
                  child: const Text('稍后'),
                ),
              FilledButton(
                onPressed: () async {
                  await _updateService.openUpdateLink(update.apkUrl);
                },
                child: Text(update.forceUpdate ? '立即更新(必需)' : '立即更新'),
              ),
            ],
          ),
        );
      },
    );
  }

  Future<void> _loginWithApiKey() async {
    if (!_apiKeyFormKey.currentState!.validate()) return;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      final state = context.read<AppState>();
      await state.loginWithApiKey(
        apiUrl: _apiUrlController.text.trim(),
        apiKey: _apiKeyController.text.trim(),
        username: _usernameController.text.trim().isEmpty
            ? 'Admin'
            : _usernameController.text.trim(),
      );
    } catch (e) {
      _error = _mapLoginError(e.toString());
    } finally {
      if (mounted) {
        setState(() {
          _saving = false;
        });
      }
    }
  }

  Future<void> _loginWithPassword() async {
    if (!_passwordFormKey.currentState!.validate()) return;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      final apiUrl = _apiUrlController.text.trim();
      final auth = AdminAuthService();
      final tokens = await auth.login(
        apiUrl: apiUrl,
        username: _loginUserController.text.trim(),
        password: _loginPasswordController.text.trim(),
      );
      final client = ApiClient(baseUrl: apiUrl, token: tokens.accessToken);
      final profile = await client.getJson('/admin/api/v1/profile');
      final username = (profile['username'] as String?)?.trim();
      final email = profile['email'] as String?;
      await context.read<AppState>().loginWithPassword(
        apiUrl: apiUrl,
        tokens: tokens,
        username: username?.isNotEmpty == true ? username! : 'Admin',
        email: email,
      );
    } on AuthException catch (e) {
      _error = _mapLoginError(e.message);
    } catch (e) {
      _error = _mapLoginError(e.toString());
    } finally {
      if (mounted) {
        setState(() {
          _saving = false;
        });
      }
    }
  }

  String _mapLoginError(String raw) {
    final message = raw.toLowerCase();
    if (message.contains('invalid credentials')) {
      return '无效登录凭据';
    }
    return '登录失败: $raw';
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      body: SafeArea(
        child: Container(
          color: const Color(0xFFF5F7F6),
          child: Center(
            child: SingleChildScrollView(
              padding: const EdgeInsets.all(24),
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 560),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        const _AppLogo(),
                        const SizedBox(width: 12),
                        Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              'Admin Portal',
                              style: theme.textTheme.titleLarge?.copyWith(
                                fontWeight: FontWeight.w700,
                              ),
                            ),
                            const SizedBox(height: 4),
                            Text(
                              'Cloud Management',
                              style: theme.textTheme.bodySmall?.copyWith(
                                color: theme.colorScheme.onSurfaceVariant,
                              ),
                            ),
                          ],
                        ),
                      ],
                    ),
                    const SizedBox(height: 12),
                    Card(
                      child: Padding(
                        padding: const EdgeInsets.all(20),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              '登录以继续',
                              style: theme.textTheme.titleMedium?.copyWith(
                                fontWeight: FontWeight.w700,
                              ),
                            ),
                            const SizedBox(height: 12),
                            TextFormField(
                              controller: _apiUrlController,
                              decoration: const InputDecoration(
                                labelText: 'API 地址',
                                hintText: 'https://api.example.com',
                              ),
                              validator: (value) {
                                if (value == null || value.trim().isEmpty) {
                                  return '请填写 API 地址';
                                }
                                return null;
                              },
                            ),
                            const SizedBox(height: 16),
                            TabBar(
                              controller: _tabController,
                              labelColor: theme.colorScheme.primary,
                              unselectedLabelColor: Colors.black54,
                              tabs: const [
                                Tab(text: '密码登录'),
                                Tab(text: 'API Key'),
                              ],
                            ),
                            const SizedBox(height: 16),
                            SizedBox(
                              height: 230,
                              child: TabBarView(
                                controller: _tabController,
                                children: [
                                  Form(
                                    key: _passwordFormKey,
                                    child: Column(
                                      children: [
                                        TextFormField(
                                          controller: _loginUserController,
                                          decoration: const InputDecoration(
                                            labelText: '用户名',
                                          ),
                                          validator: (value) {
                                            if (value == null ||
                                                value.trim().isEmpty) {
                                              return '请输入用户名';
                                            }
                                            return null;
                                          },
                                        ),
                                        const SizedBox(height: 12),
                                        TextFormField(
                                          controller: _loginPasswordController,
                                          obscureText: true,
                                          decoration: const InputDecoration(
                                            labelText: '密码',
                                          ),
                                          validator: (value) {
                                            if (value == null ||
                                                value.trim().isEmpty) {
                                              return '请输入密码';
                                            }
                                            return null;
                                          },
                                        ),
                                        const Spacer(),
                                        SizedBox(
                                          width: double.infinity,
                                          child: FilledButton(
                                            onPressed: _saving
                                                ? null
                                                : _loginWithPassword,
                                            child: _saving
                                                ? const SizedBox(
                                                    height: 20,
                                                    width: 20,
                                                    child:
                                                        CircularProgressIndicator(
                                                          strokeWidth: 2,
                                                        ),
                                                  )
                                                : const Text('登录'),
                                          ),
                                        ),
                                      ],
                                    ),
                                  ),
                                  Form(
                                    key: _apiKeyFormKey,
                                    child: Column(
                                      children: [
                                        TextFormField(
                                          controller: _apiKeyController,
                                          decoration: const InputDecoration(
                                            labelText: 'API Key',
                                            hintText: 'ak_live_xxx',
                                          ),
                                          validator: (value) {
                                            if (value == null ||
                                                value.trim().isEmpty) {
                                              return '请输入 API Key';
                                            }
                                            return null;
                                          },
                                        ),
                                        const SizedBox(height: 12),
                                        TextFormField(
                                          controller: _usernameController,
                                          decoration: const InputDecoration(
                                            labelText: '展示名称',
                                            hintText: '管理员',
                                          ),
                                        ),
                                        const Spacer(),
                                        SizedBox(
                                          width: double.infinity,
                                          child: FilledButton(
                                            onPressed: _saving
                                                ? null
                                                : _loginWithApiKey,
                                            child: _saving
                                                ? const SizedBox(
                                                    height: 20,
                                                    width: 20,
                                                    child:
                                                        CircularProgressIndicator(
                                                          strokeWidth: 2,
                                                        ),
                                                  )
                                                : const Text('登录'),
                                          ),
                                        ),
                                      ],
                                    ),
                                  ),
                                ],
                              ),
                            ),
                            if (_error != null) ...[
                              const SizedBox(height: 12),
                              Text(
                                _error!,
                                style: theme.textTheme.bodySmall?.copyWith(
                                  color: Colors.redAccent,
                                ),
                              ),
                            ],
                          ],
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _AppLogo extends StatelessWidget {
  const _AppLogo();

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(12),
      child: Image.asset(
        'assets/admin_logo.png',
        width: 56,
        height: 56,
        fit: BoxFit.cover,
      ),
    );
  }
}
