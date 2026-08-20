import 'package:flutter/material.dart';

import 'ink_seal.dart';

ThemeData inkSealTheme(InkSeal c, {required Brightness brightness}) {
  final scheme = ColorScheme(
    brightness: brightness,
    primary: c.btn,
    onPrimary: c.btnFg,
    secondary: c.seal,
    onSecondary: c.carve,
    tertiary: c.seal,
    onTertiary: c.carve,
    error: c.fgDanger,
    onError: c.carve,
    surface: c.surfaceRaised,
    onSurface: c.fgPrimary,
    onSurfaceVariant: c.fgSecondary,
    outline: c.strokeSubtle,
    outlineVariant: c.strokeStrong,
    surfaceContainerHighest: c.surfaceSunken,
    primaryContainer: c.surfaceSunken,
    onPrimaryContainer: c.fgPrimary,
    secondaryContainer: c.brandSubtle,
    onSecondaryContainer: c.sealStrong,
  );

  final shapeControl = RoundedRectangleBorder(
    borderRadius: BorderRadius.circular(InkSeal.radiusControl),
  );
  final shapeCard = RoundedRectangleBorder(
    borderRadius: BorderRadius.circular(InkSeal.radiusCard),
  );

  final text = _textTheme(c);

  return ThemeData(
    useMaterial3: true,
    brightness: brightness,
    colorScheme: scheme,
    scaffoldBackgroundColor: c.surfaceBase,
    canvasColor: c.surfaceBase,
    fontFamily: InkSeal.sans,
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
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
        textStyle: text.labelLarge,
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: c.fgPrimary,
        side: BorderSide(color: c.strokeStrong),
        shape: shapeControl,
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(foregroundColor: c.fgBrand),
    ),
    floatingActionButtonTheme: FloatingActionButtonThemeData(
      backgroundColor: c.btn,
      foregroundColor: c.btnFg,
      shape: const CircleBorder(),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: c.surfaceSunken,
      hintStyle: TextStyle(color: c.fgMuted),
      labelStyle: TextStyle(color: c.fgSecondary),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(InkSeal.radiusControl),
        borderSide: BorderSide(color: c.strokeSubtle),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(InkSeal.radiusControl),
        borderSide: BorderSide(color: c.strokeSubtle),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(InkSeal.radiusControl),
        borderSide: BorderSide(color: c.seal, width: 1.5),
      ),
    ),
    dividerColor: c.strokeSubtle,
    snackBarTheme: SnackBarThemeData(
      backgroundColor: c.fgPrimary,
      contentTextStyle: TextStyle(color: c.fgInverse),
    ),
  );
}

TextTheme _textTheme(InkSeal c) {
  TextStyle serif(double size, FontWeight weight) => TextStyle(
        fontFamily: InkSeal.serif,
        fontSize: size,
        fontWeight: weight,
        height: 1.15,
        color: c.fgPrimary,
      );
  TextStyle sans(double size, FontWeight weight, {Color? color, double height = 1.5}) => TextStyle(
        fontFamily: InkSeal.sans,
        fontSize: size,
        fontWeight: weight,
        height: height,
        color: color ?? c.fgPrimary,
      );
  return TextTheme(
    displaySmall: serif(40, FontWeight.w700),
    headlineMedium: serif(28, FontWeight.w700),
    headlineSmall: serif(22, FontWeight.w700),
    titleLarge: serif(20, FontWeight.w700),
    titleMedium: sans(16, FontWeight.w600),
    titleSmall: sans(14, FontWeight.w600),
    bodyLarge: sans(16, FontWeight.w400, color: c.fgSecondary),
    bodyMedium: sans(14, FontWeight.w400, color: c.fgSecondary),
    bodySmall: sans(12, FontWeight.w400, color: c.fgMuted),
    labelLarge: sans(14, FontWeight.w600, color: c.btnFg, height: 1.2),
  );
}
