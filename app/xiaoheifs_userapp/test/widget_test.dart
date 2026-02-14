import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:opensource_userapp/main.dart';
import 'package:opensource_userapp/core/storage/storage_service.dart';

void main() {
  testWidgets('App builds without crashing', (WidgetTester tester) async {
    TestWidgetsFlutterBinding.ensureInitialized();
    SharedPreferences.setMockInitialValues({});
    await StorageService.init();

    await tester.pumpWidget(
      const ProviderScope(
        child: MyApp(),
      ),
    );

    await tester.pump();
    expect(find.byType(MyApp), findsOneWidget);
  });
}
