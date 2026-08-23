import 'dart:js_interop';

import 'package:web/web.dart';

// CanvasKit paints the first frame with whatever glyphs are available;
// the Google Fonts CJK subsets arrive a second or two later, so an
// ungated first frame flashes tofu boxes. The bootstrap script is async
// and is NOT blocked on the fonts stylesheet, so the @font-face rules may
// still be unparsed when Dart runs and a single fonts.load() call can
// resolve empty. Poll load()+check() until every face the theme uses is
// actually available, turning the flash into blank bootstrap time.
const _faces = [
  '400 16px "Noto Sans SC"',
  '500 16px "Noto Sans SC"',
  '600 16px "Noto Sans SC"',
  '700 16px "Noto Sans SC"',
  '600 16px "Noto Serif SC"',
  '700 16px "Noto Serif SC"',
];

// Most common hanzi plus CJK punctuation: pre-warms the slices the app's
// screens actually use, not just the landing hero.
const _sample =
    '的一是了我不人在他有这上们来到时大地为子中你说生国年着就那和要她出也得里后自以会家可下而过天去能对小多然于心学么之都好看起发当没成只如事把还用第样道想作种开美总从无情己面最女但现前些所同日手又行意动，。、？！：；（）';

Future<void> loadWebFonts() async {
  final fonts = document.fonts;
  bool ready() => _faces.every((f) => fonts.check(f, _sample));
  final deadline = DateTime.now().add(const Duration(seconds: 6));
  while (!ready() && DateTime.now().isBefore(deadline)) {
    for (final f in _faces) {
      await fonts
          .load(f, _sample)
          .toDart
          .then((_) {}, onError: (_) {});
    }
    if (ready()) break;
    await Future<void>.delayed(const Duration(milliseconds: 120));
  }
}
