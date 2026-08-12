import 'dart:async';
import 'dart:io';
import 'desktop_launcher.dart';

/// 校验远程主机地址，仅允许 hostname / IPv4 / IPv6 字面量，
/// 拒绝 `;`、`&`、`|`、`` ` ``、`$()` 等 shell 元字符，防止命令注入。
bool isSafeRemoteHost(String host) {
  return RegExp(r'^[A-Za-z0-9._\-\[\]:]+$').hasMatch(host);
}

/// 校验远程登录用户名，仅允许字母、数字、点、下划线与连字符。
bool isSafeRemoteUsername(String username) {
  return RegExp(r'^[A-Za-z0-9._\-]+$').hasMatch(username);
}

/// 校验远程端口，仅允许 1-65535 的数字。
bool isSafeRemotePort(String port) {
  final value = int.tryParse(port);
  return value != null && value >= 1 && value <= 65535;
}

class _DesktopLauncherIo implements DesktopLauncher {
  @override
  Future<void> launchWindowsRdp({
    required String host,
    required String port,
    required String username,
    required String password,
  }) async {
    // 与 SSH 分支一致：先校验 host/port/username，非法输入直接抛
    // ArgumentError（不进入 Process 层）。
    if (!isSafeRemoteHost(host)) {
      throw ArgumentError('无效的主机地址');
    }
    if (!isSafeRemoteUsername(username)) {
      throw ArgumentError('无效的用户名');
    }
    if (!isSafeRemotePort(port)) {
      throw ArgumentError('无效的端口');
    }
    // 不使用 runInShell：参数以数组形式交给 CreateProcess，避免密码中的
    // `&`、空格等被 cmd.exe 解析破坏参数边界或形成命令注入。
    final target = 'TERMSRV/$host:$port';
    await Process.run('cmdkey', [
      '/generic:$target',
      '/user:$username',
      '/pass:$password',
    ]);
    try {
      final mstsc = await Process.start('mstsc', ['/v:$host:$port']);
      // 连接结束后删除凭据：VPS 系统密码不应长期驻留 Windows 凭据管理器
      //（用户态进程/备份可读）。mstsc 退出（窗口关闭）即触发清理；启动失败
      // 或读取异常也走 finally 删除。
      unawaited(mstsc.exitCode.then((_) => _deleteRdpCredential(target)));
    } catch (_) {
      await _deleteRdpCredential(target);
      rethrow;
    }
  }

  /// 删除 cmdkey 中保存的 TERMSRV 凭据；失败不掩盖已建立的连接。
  Future<void> _deleteRdpCredential(String target) async {
    try {
      await Process.run('cmdkey', ['/delete:$target']);
    } catch (_) {
      // 删除失败仅记录，不影响连接结果。
    }
  }

  @override
  Future<void> launchWindowsSsh({
    required String host,
    required String port,
    required String username,
  }) async {
    if (!isSafeRemoteHost(host)) {
      throw ArgumentError('无效的主机地址');
    }
    if (!isSafeRemoteUsername(username)) {
      throw ArgumentError('无效的用户名');
    }
    if (!isSafeRemotePort(port)) {
      throw ArgumentError('无效的端口');
    }
    final command = 'ssh $username@$host -p $port';
    await Process.start('powershell', [
      '-NoExit',
      '-Command',
      command,
    ], runInShell: true);
  }
}

final DesktopLauncher desktopLauncher = _DesktopLauncherIo();
