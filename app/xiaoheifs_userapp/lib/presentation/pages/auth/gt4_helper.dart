import 'dart:async';

import 'package:flutter/services.dart';
import 'package:gt4_flutter_plugin/gt4_flutter_plugin.dart';
import 'package:gt4_flutter_plugin/gt4_session_configuration.dart';

class GeeTestResult {
  final bool passed;
  final bool canceled;
  final String lotNumber;
  final String captchaOutput;
  final String passToken;
  final String genTime;
  final String errorCode;
  final String message;

  const GeeTestResult({
    required this.passed,
    this.canceled = false,
    this.lotNumber = '',
    this.captchaOutput = '',
    this.passToken = '',
    this.genTime = '',
    this.errorCode = '',
    this.message = '',
  });

  static const GeeTestResult empty = GeeTestResult(passed: false);
}

Future<GeeTestResult> runGeeTestChallenge(String captchaId) async {
  final id = captchaId.trim();
  if (id.isEmpty) {
    return const GeeTestResult(passed: false, message: 'captcha_id 为空');
  }

  final completer = Completer<GeeTestResult>();

  void finish(GeeTestResult value) {
    if (!completer.isCompleted) {
      completer.complete(value);
    }
  }

  try {
    try {
      await SystemChannels.textInput.invokeMethod('TextInput.hide');
    } catch (_) {}

    final config = GT4SessionConfiguration()
      ..language = 'zho'
      ..canceledOnTouchOutside = true
      ..timeout = 10000;

    final plugin = Gt4FlutterPlugin(id, config);

    plugin.addEventHandler(
      onShow: (_) {
        // No-op: keep for alignment with official usage.
      },
      onResult: (event) {
        final status = (event['status'] ?? '').toString();
        final raw = event['result'];
        final data = raw is Map ? raw : <dynamic, dynamic>{};

        final lotNumber = (data['lot_number'] ?? '').toString();
        final captchaOutput = (data['captcha_output'] ?? '').toString();
        final passToken = (data['pass_token'] ?? '').toString();
        final genTime = (data['gen_time'] ?? '').toString();

        final passed =
            status == '1' &&
            lotNumber.isNotEmpty &&
            captchaOutput.isNotEmpty &&
            passToken.isNotEmpty &&
            genTime.isNotEmpty;

        if (passed) {
          finish(
            GeeTestResult(
              passed: true,
              lotNumber: lotNumber,
              captchaOutput: captchaOutput,
              passToken: passToken,
              genTime: genTime,
            ),
          );
          return;
        }

        finish(const GeeTestResult(passed: false, message: '验证未通过'));
      },
      onError: (event) {
        final code = (event['code'] ?? '').toString();
        final msg = (event['msg'] ?? '').toString();
        // -14460: User cancelled 'Captcha'
        final canceled = code == '-14460';
        finish(
          GeeTestResult(
            passed: false,
            canceled: canceled,
            errorCode: code,
            message: msg,
          ),
        );
      },
    );

    await Future.delayed(const Duration(milliseconds: 80));
    plugin.verify();
  } on MissingPluginException {
    throw Exception('GeeTest 插件未注册，请执行完整重启后再试');
  } catch (e) {
    return GeeTestResult(passed: false, message: e.toString());
  }

  return completer.future.timeout(
    const Duration(seconds: 30),
    onTimeout: () => const GeeTestResult(passed: false, message: '验证超时'),
  );
}
