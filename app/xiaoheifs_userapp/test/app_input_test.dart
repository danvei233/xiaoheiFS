import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:opensource_userapp/presentation/widgets/common/app_input.dart';

/// F9 回归测试：密码输入框必须关闭 autocorrect / enableSuggestions，
/// 避免第三方键盘学习并联想口令片段。
void main() {
  testWidgets('AppInput 密码框关闭 autocorrect 与 suggestions', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(body: AppInput(obscureText: true, hint: '密码')),
      ),
    );

    final field = tester.widget<TextFormField>(find.byType(TextFormField));
    expect(field.autocorrect, isFalse);
    expect(field.enableSuggestions, isFalse);
  });

  testWidgets('AppInput 普通输入框保持 autocorrect 与 suggestions', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(body: AppInput(hint: '账号')),
      ),
    );

    final field = tester.widget<TextFormField>(find.byType(TextFormField));
    expect(field.autocorrect, isTrue);
    expect(field.enableSuggestions, isTrue);
  });
}
