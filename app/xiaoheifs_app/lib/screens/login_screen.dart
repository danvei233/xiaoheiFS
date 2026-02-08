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
  final _usernameController = TextEditingController(text: '管理�?);

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
            ? '管理�?
            : _usernameController.text.trim(),
      );
    } catch (e) {
      _error = '登录失败�?e';
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
            username: username?.isNotEmpty == true ? username! : '����Ա',
            email: email,
          );
    } on AuthException catch (e) {
      _error = '��¼ʧ�ܣ�';
    } catch (e) {
      _error = '��¼ʧ�ܣ�';
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
                              '\u8d22\u52a1\u7ba1\u7406',
                              style: theme.textTheme.titleLarge?.copyWith(
                                fontWeight: FontWeight.w700,
                              ),
                            ),
                            const SizedBox(height: 4),
                            Text(
                              '\u8d22\u52a1\u540e\u53f0\u7ba1\u7406',
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
                              '管理员登�?,
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
                                  return '请输�?API 地址';
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
                                            labelText: '管理员账�?,
                                          ),
                                          validator: (value) {
                                            if (value == null ||
                                                value.trim().isEmpty) {
                                              return '请输入账�?;
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
                                              return '请输入密�?;
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
                                              return '请输�?API Key';
                                            }
                                            return null;
                                          },
                                        ),
                                        const SizedBox(height: 12),
                                        TextFormField(
                                          controller: _usernameController,
                                          decoration: const InputDecoration(
                                            labelText: '显示�?,
                                            hintText: '管理�?,
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
    final colorScheme = Theme.of(context).colorScheme;
    return Container(
      width: 56,
      height: 56,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(16),
        gradient: const LinearGradient(
          colors: [Color(0xFF1E88E5), Color(0xFF42A5F5)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        boxShadow: [
          BoxShadow(
            color: colorScheme.shadow.withOpacity(0.12),
            blurRadius: 10,
            offset: const Offset(0, 4),
          ),
        ],
      ),
      child: Stack(
        alignment: Alignment.center,
        children: [
          const Text(
            '\u8d22',
            style: TextStyle(
              color: Colors.white,
              fontSize: 24,
              fontWeight: FontWeight.w800,
            ),
          ),
          Positioned(
            right: 10,
            bottom: 10,
            child: Container(
              width: 16,
              height: 16,
              decoration: BoxDecoration(
                color: Colors.white.withOpacity(0.18),
                borderRadius: BorderRadius.circular(6),
              ),
              child: const Icon(
                Icons.trending_up,
                size: 12,
                color: Colors.white,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

