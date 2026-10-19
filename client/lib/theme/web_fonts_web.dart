// The web boot has no font gate (rv4-FR-F14, F215, F210, D01; residue R3-30).
//
// loadWebFonts() used to hold the boot loading page until document.fonts.load()
// reported the CSS faces (Archivo, Noto Sans SC, Source Serif 4, Noto Serif SC)
// that a Google Fonts stylesheet in web/index.html declared. rv4-COMP-03 took
// that stylesheet out, and CanvasKit never read CSS @font-face anyway: it paints
// with the fonts it downloads itself, from web/fonts/s through the
// fontFallbackBaseUrl in web/flutter_bootstrap.js. With no face declared,
// load() can only resolve empty, so the poll ran to its 10 s deadline on every
// cold start and waited for nothing: the engine asked for the Chinese slices
// only once the gate had lifted, so nothing was warmed either.
//
// Boot now waits on session and appearance alone. If a stylesheet that declares
// faces ever returns to web/, gate on those faces here; web_font_assets_test.dart
// fails when one appears.
Future<void> loadWebFonts() async {}
