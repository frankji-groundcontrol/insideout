import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

/// 国风留白 / Ink & Seal semantic colors.
///
/// Channel values match the last live `app/src/assets/tokens.css` (git
/// `f897fb4`) and [`DESIGN.md`](../../../DESIGN.md). Primary action is ink,
/// not vermilion.
class InkSeal {
  const InkSeal({
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
    required this.seal,
    required this.sealStrong,
    required this.sealLocked,
    required this.carve,
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
  final Color seal;
  final Color sealStrong;
  final Color sealLocked;
  final Color carve;
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

  static const radiusControl = 10.0;
  static const radiusCard = 16.0;
  static const radiusHero = 24.0;

  /// Web resolves these family names through index.html's Google Fonts
  /// link; every other target (iOS, Android, desktop) loads the bundled
  /// variable fonts registered in pubspec.yaml under the *Native names,
  /// so the web bundle stays free of ~40 MB of font assets.
  static String get sans => kIsWeb ? 'Noto Sans SC' : sansNative;
  static String get serif => kIsWeb ? 'Noto Serif SC' : serifNative;

  static const sansNative = 'NotoSansSC';
  static const serifNative = 'NotoSerifSC';

  static const light = InkSeal(
    brand: Color(0xFFC8402F),
    brandHover: Color(0xFFA83426),
    brandSubtle: Color(0xFFF2DDD7),
    surfaceBase: Color(0xFFE4EAE4),
    surfaceRaised: Color(0xFFEEF1EB),
    surfaceSunken: Color(0xFFD8E0D8),
    strokeSubtle: Color(0xFFC2CABF),
    strokeStrong: Color(0xFFA9B3A5),
    fgPrimary: Color(0xFF211E1A),
    fgSecondary: Color(0xFF46433C),
    fgMuted: Color(0xFF5E5B52),
    fgInverse: Color(0xFFF2F4EE),
    fgBrand: Color(0xFFC8402F),
    fgDanger: Color(0xFFB5362A),
    seal: Color(0xFFC8402F),
    sealStrong: Color(0xFFA83426),
    sealLocked: Color(0xFF8C8A80),
    carve: Color(0xFFF2F4EE),
    btn: Color(0xFF211E1A),
    btnFg: Color(0xFFF2F4EE),
    statusNeutralBg: Color(0xFFDDE0DA),
    statusNeutralFg: Color(0xFF5E5B52),
    statusInfoBg: Color(0xFFF2DED9),
    statusInfoFg: Color(0xFFA8362A),
    statusWarnBg: Color(0xFFF3EAD6),
    statusWarnFg: Color(0xFF8A6A20),
    statusSuccessBg: Color(0xFFDCE8DC),
    statusSuccessFg: Color(0xFF3E6B4A),
  );

  static const dark = InkSeal(
    brand: Color(0xFFD84A31),
    brandHover: Color(0xFFC8402F),
    brandSubtle: Color(0xFF3A201A),
    surfaceBase: Color(0xFF161A17),
    surfaceRaised: Color(0xFF1E241F),
    surfaceSunken: Color(0xFF121613),
    strokeSubtle: Color(0xFF2E342E),
    strokeStrong: Color(0xFF3A423A),
    fgPrimary: Color(0xFFE6E9E1),
    fgSecondary: Color(0xFFB9BEB2),
    fgMuted: Color(0xFF8C9186),
    fgInverse: Color(0xFF161A17),
    fgBrand: Color(0xFFE06A52),
    fgDanger: Color(0xFFE8857A),
    seal: Color(0xFFD84A31),
    sealStrong: Color(0xFFC8402F),
    sealLocked: Color(0xFF70746C),
    carve: Color(0xFFECE7DB),
    btn: Color(0xFFE6E9E1),
    btnFg: Color(0xFF161A17),
    statusNeutralBg: Color(0xFF2A2E2A),
    statusNeutralFg: Color(0xFFAAAEA6),
    statusInfoBg: Color(0xFF3A201A),
    statusInfoFg: Color(0xFFE8907E),
    statusWarnBg: Color(0xFF3A3220),
    statusWarnFg: Color(0xFFD8B66A),
    statusSuccessBg: Color(0xFF1E2E22),
    statusSuccessFg: Color(0xFF8FB39B),
  );
}
