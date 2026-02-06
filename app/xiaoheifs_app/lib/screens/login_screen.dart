import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';
import '../services/admin_auth.dart';
import '../services/api_client.dart';
import '../services/app_storage.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen>
    with SingleTickerProviderStateMixin {
  final _apiKeyFormKey = GlobalKey<FormState>();
  final _passwordFormKey = GlobalKey<FormState>();

  final _apiUrlController = TextEditingController();
  final _apiKeyController = TextEditingController();
  final _usernameController = TextEditingController(text: '管理员');

  final _loginUserController = TextEditingController(text: 'admin');
  final _loginPasswordController = TextEditingController();

  late final TabController _tabController;
  bool _saving = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 2, vsync: this);
    _loadLastSession();
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
            ? '管理员'
            : _usernameController.text.trim(),
      );
    } catch (e) {
      _error = '登录失败：$e';
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
      final token = await auth.login(
        apiUrl: apiUrl,
        username: _loginUserController.text.trim(),
        password: _loginPasswordController.text.trim(),
      );
      final client = ApiClient(baseUrl: apiUrl, token: token);
      final profile = await client.getJson('/admin/api/v1/profile');
      final username = (profile['username'] as String?)?.trim();
      final email = profile['email'] as String?;
      await context.read<AppState>().loginWithPassword(
            apiUrl: apiUrl,
            token: token,
            username: username?.isNotEmpty == true ? username! : '管理员',
            email: email,
          );
    } on AuthException catch (e) {
      _error = '登录失败：${e.message}';
    } catch (e) {
      _error = '登录失败：$e';
    } finally {
      if (mounted) {
        setState(() {
          _saving = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      body: SafeArea(
        child: Container(
          decoration: const BoxDecoration(
            gradient: LinearGradient(
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
              colors: [
                Color(0xFFF5F9F7),
                Color(0xFFE9F3F1),
              ],
            ),
          ),
          child: Center(
            child: SingleChildScrollView(
              padding: const EdgeInsets.all(24),
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 560),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      '小黑云财务管理系统',
                      style: theme.textTheme.headlineMedium?.copyWith(
                        fontWeight: FontWeight.w800,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      '管理员控制台',
                      style: theme.textTheme.titleLarge?.copyWith(
                        color: Colors.black54,
                      ),
                    ),
                    const SizedBox(height: 24),
                    Card(
                      child: Padding(
                        padding: const EdgeInsets.all(20),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              '登录',
                              style: theme.textTheme.titleMedium?.copyWith(
                                fontWeight: FontWeight.w700,
                              ),
                            ),
                            const SizedBox(height: 12),
                            TextFormField(
                              controller: _apiUrlController,
                              decoration: const InputDecoration(
                                labelText: 'API 地址',
                                hintText: 'http://127.0.0.1:8080',
                              ),
                              validator: (value) {
                                if (value == null || value.trim().isEmpty) {
                                  return '请输入 API 地址';
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
                                Tab(text: '账号登录'),
                                Tab(text: 'API Key'),
                              ],
                            ),
                            const SizedBox(height: 16),
                            SizedBox(
                              height: 260,
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
                                            labelText: '管理员账号',
                                          ),
                                          validator: (value) {
                                            if (value == null ||
                                                value.trim().isEmpty) {
                                              return '请输入账号';
                                            }
                                            return null;
                                          },
                                        ),
                                        const SizedBox(height: 16),
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
                                            onPressed:
                                                _saving ? null : _loginWithPassword,
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
                                        const SizedBox(height: 16),
                                        TextFormField(
                                          controller: _usernameController,
                                          decoration: const InputDecoration(
                                            labelText: '显示名',
                                            hintText: '管理员',
                                          ),
                                        ),
                                        const Spacer(),
                                        SizedBox(
                                          width: double.infinity,
                                          child: FilledButton(
                                            onPressed:
                                                _saving ? null : _loginWithApiKey,
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
                    const SizedBox(height: 12),
                    Text(
                      '建议使用账号登录，自动获取权限与资料。',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: Colors.black45,
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
