import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'app.dart';
import 'router.dart';
import 'session/appearance.dart';
import 'session/boot.dart';
import 'session/session.dart';
import 'theme/native_fonts.dart';
import 'theme/web_fonts_stub.dart'
    if (dart.library.ui_web) 'theme/web_fonts_web.dart';
import 'web_url_strategy_stub.dart'
    if (dart.library.ui_web) 'web_url_strategy_web.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  configureUrlStrategy();
  final session = Session();
  final appearance = Appearance(store: SecureSettingsStore());
  final boot = BootController();
  // Hydration keeps running in the background: the app renders
  // immediately behind the boot loading page instead of a blank screen,
  // and the gate lifts (success or error) when everything has settled.
  unawaited(
    Future.wait([
      session.hydrate(),
      appearance.hydrate(),
      loadNativeFonts(),
      loadWebFonts(),
    ]).then((_) => boot.complete(), onError: (_) => boot.complete()),
  );
  runApp(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider.value(value: appearance),
        ChangeNotifierProvider.value(value: boot),
      ],
      child: InsideOutApp(router: buildRouter(session)),
    ),
  );
}
