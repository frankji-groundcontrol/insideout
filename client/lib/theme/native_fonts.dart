import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';

import 'cobalt.dart';

/// Pubspec-registered fonts are fetched eagerly on every platform,
/// including web — ~43 MB the hosted app must never download. The Noto
/// files therefore ship as plain (lazily fetched) assets, registered at
/// startup on non-web targets only; web keeps index.html's CDN fonts.
/// Family names pair with [Cobalt.serifNative] / [Cobalt.sansNative].
Future<void> loadNativeFonts() async {
  if (kIsWeb) return;
  await Future.wait([
    _load(Cobalt.serifNative, 'assets/fonts/NotoSerifSC-Var.ttf'),
    _load(Cobalt.sansNative, 'assets/fonts/NotoSansSC-Var.ttf'),
  ]);
}

Future<void> _load(String family, String asset) async {
  final loader = FontLoader(family)..addFont(rootBundle.load(asset));
  await loader.load();
}
