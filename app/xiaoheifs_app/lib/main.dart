import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:provider/provider.dart';

import 'app_state.dart';
import 'screens/login_screen.dart';
import 'screens/root_scaffold.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  if (!kIsWeb) {
    await SystemChrome.setEnabledSystemUIMode(
      SystemUiMode.manual,
      overlays: SystemUiOverlay.values,
    );
  }
  // Firebase disabled for open-source build.
  final appState = AppState();
  await appState.load();
  runApp(MyApp(appState: appState));
}

class MyApp extends StatelessWidget {
  final AppState appState;

  const MyApp({super.key, required this.appState});

  @override
  Widget build(BuildContext context) {
    return ChangeNotifierProvider.value(
      value: appState,
      child: MaterialApp(
        debugShowCheckedModeBanner: false,
        title: '\u4e91\u4eab\u4e92\u8054\u7ba1\u7406\u7aef',
        builder: (context, child) {
          final media = MediaQuery.of(context);
          final baseChild = child ?? const SizedBox.shrink();

          if (defaultTargetPlatform != TargetPlatform.android) {
            return MediaQuery(
              data: media.copyWith(textScaler: const TextScaler.linear(1.0)),
              child: baseChild,
            );
          }

          const scale = 0.70;
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
          colorScheme: ColorScheme.fromSeed(
            seedColor: const Color(0xFF1E88E5),
            brightness: Brightness.light,
          ),
          textTheme: GoogleFonts.notoSansScTextTheme(),
          scaffoldBackgroundColor: const Color(0xFFF4F7FB),
          appBarTheme: AppBarTheme(
            backgroundColor: const Color(0xFFF4F7FB),
            foregroundColor: const Color(0xFF0F172A),
            elevation: 0,
            scrolledUnderElevation: 0,
            titleTextStyle: GoogleFonts.notoSansSc(
              fontSize: 18,
              fontWeight: FontWeight.w700,
              color: const Color(0xFF0F172A),
            ),
          ),
          cardTheme: CardThemeData(
            color: Colors.white,
            surfaceTintColor: Colors.transparent,
            elevation: 0.5,
            shadowColor: const Color(0x1A0F172A),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(14),
              side: const BorderSide(color: Color(0xFFE5EAF2)),
            ),
          ),
          inputDecorationTheme: InputDecorationTheme(
            isDense: true,
            filled: true,
            fillColor: Colors.white,
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 12,
              vertical: 10,
            ),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(10),
              borderSide: const BorderSide(color: Color(0xFFD7DFEB)),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(10),
              borderSide: const BorderSide(color: Color(0xFFD7DFEB)),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(10),
              borderSide: const BorderSide(
                color: Color(0xFF1E88E5),
                width: 1.2,
              ),
            ),
          ),
          filledButtonTheme: FilledButtonThemeData(
            style: FilledButton.styleFrom(
              elevation: 0,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(10),
              ),
              padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
            ),
          ),
          outlinedButtonTheme: OutlinedButtonThemeData(
            style: OutlinedButton.styleFrom(
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(10),
              ),
              side: const BorderSide(color: Color(0xFFD7DFEB)),
              padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
            ),
          ),
          snackBarTheme: SnackBarThemeData(
            backgroundColor: const Color(0xFF0F172A),
            contentTextStyle: GoogleFonts.notoSansSc(color: Colors.white),
            behavior: SnackBarBehavior.floating,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(10),
            ),
          ),
          listTileTheme: const ListTileThemeData(
            minLeadingWidth: 34,
            horizontalTitleGap: 10,
            visualDensity: VisualDensity.compact,
          ),
          tabBarTheme: TabBarThemeData(
            labelStyle: GoogleFonts.notoSansSc(
              fontSize: 12,
              fontWeight: FontWeight.w600,
            ),
            unselectedLabelStyle: GoogleFonts.notoSansSc(
              fontSize: 12,
              fontWeight: FontWeight.w500,
            ),
            dividerColor: Colors.transparent,
          ),
          dialogTheme: DialogThemeData(
            backgroundColor: Colors.white,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(14),
            ),
          ),
          bottomSheetTheme: const BottomSheetThemeData(
            backgroundColor: Colors.white,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
            ),
          ),
          floatingActionButtonTheme: const FloatingActionButtonThemeData(
            backgroundColor: Color(0xFF1E88E5),
            foregroundColor: Colors.white,
          ),
          useMaterial3: true,
        ),
        home: Consumer<AppState>(
          builder: (context, state, _) {
            if (!state.isReady) {
              return const Scaffold(
                body: Center(child: CircularProgressIndicator()),
              );
            }
            return state.isLoggedIn
                ? const RootScaffold()
                : const LoginScreen();
          },
        ),
      ),
    );
  }
}
