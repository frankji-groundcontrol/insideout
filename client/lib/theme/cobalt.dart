import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

/// Cobalt card lock — one hue, neutrals carry the page.
///
/// Values match [`docs/plans/2026-09-18-cobalt-flutter-goal.md`](../../../docs/plans/2026-09-18-cobalt-flutter-goal.md)
/// and the lock screens in `docs/design-system/cobalt/`. Primary action is
/// cobalt-500, not ink.
class Cobalt {
  const Cobalt({
    required this.brand,
    required this.brandHover,
    required this.brandSubtle,
    required this.surfaceBase,
    required this.surfaceRaised,
    required this.surfaceSunken,
    required this.strokeSubtle,
    required this.strokeStrong,
    required this.fgPrimary,
    required this.fgSecondary,
    required this.fgMuted,
    required this.fgInverse,
    required this.fgBrand,
    required this.fgDanger,
    required this.btn,
    required this.btnFg,
    required this.statusNeutralBg,
    required this.statusNeutralFg,
    required this.statusInfoBg,
    required this.statusInfoFg,
    required this.statusWarnBg,
    required this.statusWarnFg,
    required this.statusSuccessBg,
    required this.statusSuccessFg,
  });

  final Color brand;
  final Color brandHover;
  final Color brandSubtle;
  final Color surfaceBase;
  final Color surfaceRaised;
  final Color surfaceSunken;
  final Color strokeSubtle;
  final Color strokeStrong;
  final Color fgPrimary;
  final Color fgSecondary;
  final Color fgMuted;
  final Color fgInverse;
  final Color fgBrand;
  final Color fgDanger;
  final Color btn;
  final Color btnFg;
  final Color statusNeutralBg;
  final Color statusNeutralFg;
  final Color statusInfoBg;
  final Color statusInfoFg;
  final Color statusWarnBg;
  final Color statusWarnFg;
  final Color statusSuccessBg;
  final Color statusSuccessFg;

  static const hue50 = Color(0xFFEEF2FF);
  static const hue200 = Color(0xFFC7D2FE);
  static const hue500 = Color(0xFF3B5BFF);
  static const hue700 = Color(0xFF1D4ED8);
  static const hue950 = Color(0xFF1E2A5A);
  static const lightGround = Color(0xFFF7F8FA);
  static const lightInk = Color(0xFF0B0D12);
  static const darkGround = Color(0xFF0B0D12);
  static const darkType = Color(0xFFF7F8FA);

  static const radiusControl = 12.0;
  static const radiusCard = 20.0;
  static const radiusHero = 28.0;
  static const radiusPill = 999.0;

  /// Web Latin UI face. CJK falls back to [sansCjk] so zh-CN never tofu.
  static const sansWeb = 'Archivo';
  static const sansCjkWeb = 'Noto Sans SC';
  /// Coach-only serif. Latin + CJK pair.
  static const serifWeb = 'Source Serif 4';
  static const serifCjkWeb = 'Noto Serif SC';

  /// Native registers these lazily from assets — never eager pubspec fonts.
  static const sansNative = 'NotoSansSC';
  static const serifNative = 'NotoSerifSC';

  static String get sans => kIsWeb ? sansWeb : sansNative;
  static String get sansCjk => kIsWeb ? sansCjkWeb : sansNative;
  static String get serif => kIsWeb ? serifWeb : serifNative;
  static String get serifCjk => kIsWeb ? serifCjkWeb : serifNative;

  static List<String> get sansFallback => [sansCjk];
  static List<String> get serifFallback => [serifCjk];

  /// Ideas wall display: clamp(64px, 10vw, 320px) on wide web.
  static double ideasDisplaySize(double width) => (width * 0.10).clamp(64.0, 320.0);

  static Cobalt of(BuildContext context) =>
      Theme.of(context).brightness == Brightness.dark ? dark : light;

  static const light = Cobalt(
    brand: hue500,
    brandHover: hue700,
    brandSubtle: hue50,
    surfaceBase: lightGround,
    surfaceRaised: Color(0xFFFFFFFF),
    surfaceSunken: hue50,
    strokeSubtle: hue200,
    strokeStrong: Color(0xFF9AA7E8),
    fgPrimary: lightInk,
    fgSecondary: Color(0xFF3A3F4A),
    fgMuted: Color(0xFF6B7280),
    fgInverse: darkType,
    fgBrand: hue500,
    fgDanger: hue700,
    btn: hue500,
    btnFg: Color(0xFFFFFFFF),
    statusNeutralBg: hue50,
    statusNeutralFg: Color(0xFF3A3F4A),
    statusInfoBg: hue50,
    statusInfoFg: hue700,
    statusWarnBg: hue200,
    statusWarnFg: hue700,
    statusSuccessBg: hue50,
    statusSuccessFg: hue700,
  );

  static const dark = Cobalt(
    brand: hue500,
    brandHover: hue700,
    brandSubtle: hue950,
    surfaceBase: darkGround,
    surfaceRaised: Color(0xFF141820),
    surfaceSunken: hue950,
    strokeSubtle: Color(0xFF2A3358),
    strokeStrong: Color(0xFF3B4A7A),
    fgPrimary: darkType,
    fgSecondary: Color(0xFFC5CAD3),
    fgMuted: Color(0xFF8B93A1),
    fgInverse: lightInk,
    fgBrand: hue200,
    fgDanger: hue200,
    btn: hue500,
    btnFg: darkType,
    statusNeutralBg: Color(0xFF1A1F2E),
    statusNeutralFg: Color(0xFFC5CAD3),
    statusInfoBg: hue950,
    statusInfoFg: hue200,
    statusWarnBg: hue950,
    statusWarnFg: hue200,
    statusSuccessBg: hue950,
    statusSuccessFg: hue200,
  );
}
