import 'dart:js_interop';

import 'package:web/web.dart';

// CanvasKit paints the first frame with whatever glyphs are available;
// the Google Fonts CJK subsets arrive a second or two later, so an
// ungated first frame flashes tofu boxes. The bootstrap script is async
// and is NOT blocked on the fonts stylesheet, so when Dart runs the
// @font-face rules may still be unparsed — check() then returns true
// ("no web font needed") and document.styleSheets already lists the
// <link>, so neither is a usable gate. fonts.load() is: it resolves
// EMPTY while no matching face exists and with the loaded faces once
// they are ready. Poll it until every face the theme uses reports
// loaded, turning the flash into blank bootstrap time.
const _faces = [
  '400 16px "Archivo"',
  '500 16px "Archivo"',
  '600 16px "Archivo"',
  '700 16px "Archivo"',
  '400 16px "Noto Sans SC"',
  '500 16px "Noto Sans SC"',
  '600 16px "Noto Sans SC"',
  '700 16px "Noto Sans SC"',
  '400 16px "Source Serif 4"',
  '600 16px "Source Serif 4"',
  '700 16px "Source Serif 4"',
  '400 16px "Noto Serif SC"',
  '600 16px "Noto Serif SC"',
  '700 16px "Noto Serif SC"',
];

// Most common hanzi plus CJK punctuation: pre-warms the slices the app's
// screens actually use, not just the landing hero.
const _sample =
    '的一是了我不人在他有这上们来到时大地为子中你说生国年着就那和要她出也得里后自以会家可下而过天去能对小多然于心学么之都好看起发当没成只如事把还用第样道想作种开美总从无情己面最女但现前些所同日手又行意动，。、？！：；（）';

Future<bool> _allFacesLoaded() async {
  for (final f in _faces) {
    // fonts.load() resolves empty while the face is unparsed, but it
    // never settles when the font host is unreachable — a pending await
    // here would pin the boot gate open. A timeout counts as not loaded
    // and the outer deadline gives up.
    final loaded = await document.fonts
        .load(f, _sample)
        .toDart
        .timeout(const Duration(seconds: 2), onTimeout: () => <FontFace>[].toJS)
        .catchError((_) => <FontFace>[].toJS);
    if (loaded.toDart.isEmpty) return false;
  }
  return true;
}

Future<void> loadWebFonts() async {
  final deadline = DateTime.now().add(const Duration(seconds: 10));
  while (DateTime.now().isBefore(deadline)) {
    if (await _allFacesLoaded()) return;
    await Future<void>.delayed(const Duration(milliseconds: 120));
  }
}
