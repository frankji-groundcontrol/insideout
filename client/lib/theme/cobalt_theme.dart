import 'package:flutter/material.dart';

import 'cobalt.dart';

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
    borderRadius: BorderRadius.circular(Cobalt.radiusPill),
  );
  final shapeCard = RoundedRectangleBorder(
    borderRadius: BorderRadius.circular(Cobalt.radiusCard),
  );

  final text = _textTheme(c);

  return ThemeData(
    useMaterial3: true,
    brightness: brightness,
    colorScheme: scheme,
    scaffoldBackgroundColor: c.surfaceBase,
    canvasColor: c.surfaceBase,
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
      shape: shapeCard.copyWith(
        side: BorderSide(color: c.strokeSubtle),
      ),
      margin: EdgeInsets.zero,
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: c.btn,
        foregroundColor: c.btnFg,
        shape: shapeControl,
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
        textStyle: text.labelLarge,
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: c.fgPrimary,
        side: BorderSide(color: c.strokeSubtle),
        shape: shapeControl,
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(foregroundColor: c.fgBrand),
    ),
    floatingActionButtonTheme: FloatingActionButtonThemeData(
      backgroundColor: c.btn,
      foregroundColor: c.btnFg,
      shape: const StadiumBorder(),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: c.surfaceRaised,
      hintStyle: TextStyle(color: c.fgMuted, fontSize: 17.5),
      labelStyle: TextStyle(color: c.fgSecondary, fontSize: 12),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(Cobalt.radiusPill),
        borderSide: BorderSide(color: c.strokeSubtle),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(Cobalt.radiusPill),
        borderSide: BorderSide(color: c.strokeSubtle),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(Cobalt.radiusPill),
        borderSide: const BorderSide(color: Cobalt.hue500, width: 1.5),
      ),
    ),
    chipTheme: ChipThemeData(
      backgroundColor: c.brandSubtle,
      selectedColor: c.brand,
      labelStyle: TextStyle(color: c.fgPrimary, fontSize: 12, fontWeight: FontWeight.w600),
      shape: const StadiumBorder(),
      side: BorderSide(color: c.strokeSubtle),
    ),
    dividerColor: c.strokeSubtle,
    snackBarTheme: SnackBarThemeData(
      backgroundColor: c.fgPrimary,
      contentTextStyle: TextStyle(color: c.fgInverse),
    ),
  );
}

TextTheme _textTheme(Cobalt c) {
  TextStyle sans(double size, FontWeight weight, {Color? color, double height = 1.45}) => TextStyle(
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

TextStyle cobaltSerif(Cobalt c, {double size = 26, FontWeight weight = FontWeight.w600, Color? color, double height = 1.35}) {
  return TextStyle(
    fontFamily: Cobalt.serif,
    fontFamilyFallback: Cobalt.serifFallback,
    fontSize: size,
    fontWeight: weight,
    height: height,
    color: color ?? c.fgPrimary,
  );
}
