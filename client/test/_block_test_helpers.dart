import 'package:flutter/material.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_theme.dart';
import 'package:provider/provider.dart';

Widget host(Widget child, {bool dark = true}) {
  final palette = dark ? Cobalt.dark : Cobalt.light;
  return ChangeNotifierProvider(
    create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US'),
    child: MaterialApp(
      theme: cobaltTheme(
        palette,
        brightness: dark ? Brightness.dark : Brightness.light,
      ),
      home: Scaffold(body: child),
    ),
  );
}
