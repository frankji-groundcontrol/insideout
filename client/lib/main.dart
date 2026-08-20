import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'app.dart';
import 'router.dart';
import 'session/appearance.dart';
import 'session/session.dart';
import 'theme/native_fonts.dart';
import 'web_url_strategy_stub.dart'
    if (dart.library.ui_web) 'web_url_strategy_web.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  configureUrlStrategy();
  final session = Session();
  final appearance = Appearance(store: SecureSettingsStore());
  await Future.wait([session.hydrate(), appearance.hydrate(), loadNativeFonts()]);
  runApp(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider.value(value: appearance),
      ],
      child: InsideOutApp(router: buildRouter(session)),
    ),
  );
}
