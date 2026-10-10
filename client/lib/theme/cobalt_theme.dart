import 'package:flutter/material.dart';

import 'cobalt.dart';

/// Cast shadow color — solid ink on the dark ramp, black at ~15% on light.
Color castShadowColor(Brightness brightness) => brightness == Brightness.dark
    ? const Color(0xFF05070A)
    : const Color(0x26000000);

/// Cobalt-Neo cast shadow — hard 4×4 offset, zero blur, zero spread.
List<BoxShadow> cast(Brightness brightness) => <BoxShadow>[
  BoxShadow(
    color: castShadowColor(brightness),
    offset: const Offset(4, 4),
    blurRadius: 0,
    spreadRadius: 0,
  ),
];

/// Hover-state cast shadow — hard 2×2 offset, zero blur, zero spread.
List<BoxShadow> castSm(Brightness brightness) => <BoxShadow>[
  BoxShadow(
    color: castShadowColor(brightness),
    offset: const Offset(2, 2),
    blurRadius: 0,
    spreadRadius: 0,
  ),
];

/// The uppercase eyebrow/meta label — labelSmall in the brand ink with 0.8
/// tracking. One shared recipe for every surface.
TextStyle eyebrow(BuildContext context) => Theme.of(context)
    .textTheme
    .labelSmall!
    .copyWith(color: Cobalt.of(context).fgBrand, letterSpacing: 0.8);

ThemeData cobaltTheme(Cobalt c, {required Brightness brightness}) {
  final scheme = ColorScheme(
    brightness: brightness,
    primary: c.btn,
    onPrimary: c.btnFg,
    secondary: c.brand,
    onSecondary: c.btnFg,
    tertiary: c.brandHover,
    onTertiary: c.btnFg,
    error: c.fgDanger,
    onError: c.btnFg,
    surface: c.surfaceRaised,
    onSurface: c.fgPrimary,
    onSurfaceVariant: c.fgSecondary,
    outline: c.strokeSubtle,
    outlineVariant: c.strokeStrong,
    surfaceContainerHighest: c.surfaceSunken,
    primaryContainer: c.brandSubtle,
    onPrimaryContainer: c.fgPrimary,
    secondaryContainer: c.brandSubtle,
    onSecondaryContainer: c.brandHover,
  );

  final shapeControl = RoundedRectangleBorder(
    borderRadius: BorderRadius.circular(Cobalt.radiusControl),
  );
  final shapeCard = RoundedRectangleBorder(
    borderRadius: BorderRadius.circular(Cobalt.radiusCard),
  );
  final castColor = castShadowColor(brightness);

  final text = _textTheme(c);

  return ThemeData(
    useMaterial3: true,
    brightness: brightness,
    colorScheme: scheme,
    scaffoldBackgroundColor: c.surfaceBase,
    canvasColor: c.surfaceBase,
    shadowColor: castColor,
    // The block grammar presses, it does not ripple. Material menus, pickers
    // and dropdowns that survive still read their ink from the theme, so one
    // switch here covers every one of them without per-site wrappers.
    splashFactory: NoSplash.splashFactory,
    splashColor: Colors.transparent,
    highlightColor: Colors.transparent,
    fontFamily: Cobalt.sans,
    textTheme: text,
    primaryTextTheme: text,
    appBarTheme: AppBarTheme(
      backgroundColor: c.surfaceBase,
      foregroundColor: c.fgPrimary,
      elevation: 0,
      scrolledUnderElevation: 0,
      centerTitle: false,
      titleTextStyle: text.titleLarge,
    ),
    cardTheme: CardThemeData(
      color: c.surfaceRaised,
      elevation: 0,
      shadowColor: castColor,
      shape: shapeCard.copyWith(
        side: BorderSide(color: c.strokeSubtle, width: Cobalt.borderContainer),
      ),
      margin: EdgeInsets.zero,
    ),
    dialogTheme: DialogThemeData(
      backgroundColor: c.surfaceRaised,
      elevation: 0,
      shadowColor: castColor,
      shape: shapeCard.copyWith(
        side: BorderSide(color: c.strokeSubtle, width: Cobalt.borderContainer),
      ),
    ),
    // Stock M3 menus are 4px-rounded with a blurred elevation shadow. This is
    // the block menu every PopupMenuButton adopts with no per-site styling.
    popupMenuTheme: PopupMenuThemeData(
      color: c.surfaceRaised,
      elevation: 0,
      surfaceTintColor: Colors.transparent,
      shape: shapeControl.copyWith(
        side: BorderSide(color: c.strokeSubtle, width: Cobalt.borderContainer),
      ),
    ),
    // A PopupMenuButton without a child triggers through a stock IconButton,
    // whose M3 hover/press highlight is a stadium — squared like every corner.
    iconButtonTheme: IconButtonThemeData(
      style: IconButton.styleFrom(shape: shapeControl),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: c.btn,
        foregroundColor: c.btnFg,
        shape: shapeControl,
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
        textStyle: text.labelLarge,
        animationDuration: Cobalt.pressDuration,
        elevation: 0,
        shadowColor: castColor,
      ),
    ),
    elevatedButtonTheme: ElevatedButtonThemeData(
      style: ElevatedButton.styleFrom(
        backgroundColor: c.btn,
        foregroundColor: c.btnFg,
        shape: shapeControl,
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
        textStyle: text.labelLarge,
        animationDuration: Cobalt.pressDuration,
        elevation: 0,
        shadowColor: castColor,
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: c.fgPrimary,
        side: BorderSide(color: c.strokeSubtle, width: Cobalt.borderContainer),
        shape: shapeControl,
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
        animationDuration: Cobalt.pressDuration,
        elevation: 0,
        shadowColor: castColor,
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(
        foregroundColor: c.fgBrand,
        shape: shapeControl,
        animationDuration: Cobalt.pressDuration,
        elevation: 0,
      ),
    ),
    floatingActionButtonTheme: FloatingActionButtonThemeData(
      backgroundColor: c.btn,
      foregroundColor: c.btnFg,
      shape: const RoundedRectangleBorder(),
      elevation: 0,
      hoverElevation: 0,
      focusElevation: 0,
      highlightElevation: 0,
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: c.surfaceRaised,
      hintStyle: TextStyle(color: c.fgMuted, fontSize: 17.5),
      labelStyle: TextStyle(color: c.fgSecondary, fontSize: 12),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(Cobalt.radiusControl),
        borderSide: BorderSide(
          color: c.strokeSubtle,
          width: Cobalt.borderContainer,
        ),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(Cobalt.radiusControl),
        borderSide: BorderSide(
          color: c.strokeSubtle,
          width: Cobalt.borderContainer,
        ),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(Cobalt.radiusControl),
        borderSide: const BorderSide(
          color: Cobalt.hue500,
          width: Cobalt.borderContainer,
        ),
      ),
    ),
    chipTheme: ChipThemeData(
      backgroundColor: c.brandSubtle,
      selectedColor: c.brand,
      labelStyle: TextStyle(
        color: c.fgPrimary,
        fontSize: 12,
        fontWeight: FontWeight.w600,
      ),
      shape: shapeControl,
      side: BorderSide(color: c.strokeSubtle, width: Cobalt.borderContainer),
    ),
    dividerColor: c.strokeSubtle,
    snackBarTheme: SnackBarThemeData(
      backgroundColor: c.fgPrimary,
      contentTextStyle: TextStyle(color: c.fgInverse),
      elevation: 0,
      shape: shapeCard.copyWith(
        side: BorderSide(color: c.strokeSubtle, width: Cobalt.borderContainer),
      ),
    ),
  );
}

TextTheme _textTheme(Cobalt c) {
  TextStyle sans(
    double size,
    FontWeight weight, {
    Color? color,
    double height = 1.45,
  }) => TextStyle(
    fontFamily: Cobalt.sans,
    fontFamilyFallback: Cobalt.sansFallback,
    fontSize: size,
    fontWeight: weight,
    height: height,
    color: color ?? c.fgPrimary,
  );
  return TextTheme(
    displayLarge: sans(56, FontWeight.w700, height: 0.95),
    displayMedium: sans(52, FontWeight.w700, height: 0.95),
    displaySmall: sans(52, FontWeight.w700, height: 1.0),
    headlineLarge: sans(32, FontWeight.w700, height: 1.15),
    headlineMedium: sans(28, FontWeight.w700, height: 1.15),
    headlineSmall: sans(26, FontWeight.w700, height: 1.2),
    titleLarge: sans(26, FontWeight.w700, height: 1.2),
    titleMedium: sans(17.5, FontWeight.w600, height: 1.35),
    titleSmall: sans(17.5, FontWeight.w600, height: 1.35),
    bodyLarge: sans(17.5, FontWeight.w400, color: c.fgSecondary),
    bodyMedium: sans(17.5, FontWeight.w400, color: c.fgSecondary),
    bodySmall: sans(12, FontWeight.w400, color: c.fgMuted, height: 1.4),
    labelLarge: sans(14, FontWeight.w600, color: c.btnFg, height: 1.2),
    labelMedium: sans(12, FontWeight.w600, color: c.fgSecondary, height: 1.2),
    labelSmall: sans(12, FontWeight.w600, color: c.fgMuted, height: 1.2),
  );
}

TextStyle cobaltSerif(
  Cobalt c, {
  double size = 26,
  FontWeight weight = FontWeight.w600,
  Color? color,
  double height = 1.35,
}) {
  return TextStyle(
    fontFamily: Cobalt.serif,
    fontFamilyFallback: Cobalt.serifFallback,
    fontSize: size,
    fontWeight: weight,
    height: height,
    color: color ?? c.fgPrimary,
  );
}
