import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'core/navigation/app_navigator.dart';
import 'core/network/api_client.dart';
import 'core/storage/storage_service.dart';
import 'presentation/providers/theme_provider.dart';
import 'presentation/routes/app_routes.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  await StorageService.init();

  final client = ApiClient.instance;
  final savedBaseUrl = StorageService.instance.getApiBaseUrl();
  if (savedBaseUrl != null && savedBaseUrl.isNotEmpty) {
    client.updateBaseUrl(savedBaseUrl);
  }

  runApp(const ProviderScope(child: MyApp()));
}

class MyApp extends ConsumerWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(routerProvider);
    final themeMode = ref.watch(themeModeProvider);
    final lightScheme = ColorScheme.fromSeed(
      seedColor: Colors.blue,
      brightness: Brightness.light,
    );
    final darkScheme = ColorScheme.fromSeed(
      seedColor: Colors.blue,
      brightness: Brightness.dark,
    );

    return MaterialApp.router(
      title: '云享互联',
      debugShowCheckedModeBanner: false,
      scaffoldMessengerKey: AppNavigator.messengerKey,
      builder: (context, child) {
        final media = MediaQuery.of(context);
        final baseChild = child ?? const SizedBox.shrink();

        if (defaultTargetPlatform != TargetPlatform.android) {
          return MediaQuery(
            data: media.copyWith(textScaler: const TextScaler.linear(1.0)),
            child: baseChild,
          );
        }

        const scale = 0.75;
        final scaledMedia = media.copyWith(
          textScaler: const TextScaler.linear(1.0),
          size: Size(media.size.width / scale, media.size.height / scale),
          padding: EdgeInsets.fromLTRB(
            media.padding.left / scale,
            media.padding.top / scale,
            media.padding.right / scale,
            media.padding.bottom / scale,
          ),
          viewPadding: EdgeInsets.fromLTRB(
            media.viewPadding.left / scale,
            media.viewPadding.top / scale,
            media.viewPadding.right / scale,
            media.viewPadding.bottom / scale,
          ),
          viewInsets: EdgeInsets.fromLTRB(
            media.viewInsets.left / scale,
            media.viewInsets.top / scale,
            media.viewInsets.right / scale,
            media.viewInsets.bottom / scale,
          ),
        );

        return SizedBox.expand(
          child: FittedBox(
            fit: BoxFit.fill,
            alignment: Alignment.topLeft,
            child: MediaQuery(
              data: scaledMedia,
              child: SizedBox(
                width: media.size.width / scale,
                height: media.size.height / scale,
                child: baseChild,
              ),
            ),
          ),
        );
      },
      theme: ThemeData(
        colorScheme: lightScheme,
        progressIndicatorTheme: ProgressIndicatorThemeData(
          color: lightScheme.primary,
          linearTrackColor: lightScheme.surfaceContainerHigh.withValues(
            alpha: 0.95,
          ),
          circularTrackColor: lightScheme.surfaceContainerHigh.withValues(
            alpha: 0.95,
          ),
        ),
        visualDensity: VisualDensity.compact,
        materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
        useMaterial3: true,
      ),
      darkTheme: ThemeData(
        colorScheme: darkScheme,
        progressIndicatorTheme: ProgressIndicatorThemeData(
          color: darkScheme.primary,
          linearTrackColor: darkScheme.surfaceContainerHighest.withValues(
            alpha: 0.85,
          ),
          circularTrackColor: darkScheme.surfaceContainerHighest.withValues(
            alpha: 0.85,
          ),
        ),
        visualDensity: VisualDensity.compact,
        materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
        useMaterial3: true,
      ),
      themeMode: themeMode,
      routerConfig: router,
    );
  }
}
