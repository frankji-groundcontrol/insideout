import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Wave 7.1: the web bundle must not ship the two CJK variable fonts.
/// Native platforms still list them. Web draws CJK with the fonts the engine
/// downloads itself, mirrored under web/fonts/s (web_csp_self_test.dart).
void main() {
  test('declared web assets omit the two CJK variable fonts', () {
    final pubspec = File('pubspec.yaml').readAsStringSync();
    expect(pubspec.contains('assets/fonts/OFL.txt'), isTrue);
    expect(_bundledOnWeb(pubspec, 'assets/fonts/NotoSansSC-Var.ttf'), isFalse);
    expect(_bundledOnWeb(pubspec, 'assets/fonts/NotoSerifSC-Var.ttf'), isFalse);
  });

  // rv4-FR-F14, F215, F210, D01 (R3-30): loadWebFonts() in
  // lib/theme/web_fonts_web.dart waits for nothing because no page under web/
  // declares a CSS font face. If a stylesheet returns, the gate has to return
  // with it, so this fails first.
  test('web/ declares no CSS font face for the web boot to wait on', () {
    final html = File(
      'web/index.html',
    ).readAsStringSync().replaceAll(RegExp(r'<!--.*?-->', dotAll: true), '');
    expect(
      RegExp(
        r'@font-face|<style|<link[^>]*stylesheet',
        caseSensitive: false,
      ).allMatches(html).map((m) => m.group(0)).toList(),
      isEmpty,
      reason: 'index.html declares a face again: gate loadWebFonts() on it',
    );
    final sheets = Directory('web')
        .listSync(recursive: true)
        .whereType<File>()
        .where((f) => f.path.endsWith('.css'))
        .map((f) => f.path)
        .toList();
    expect(sheets, isEmpty, reason: 'a stylesheet under web/ may declare faces');
  });
}

/// True when [assetPath] is a Flutter asset that the web target includes.
/// An entry with no `platforms` list is included everywhere. An entry whose
/// `platforms` list omits `web` is not.
bool _bundledOnWeb(String pubspec, String assetPath) {
  final lines = pubspec.split('\n');
  var inAssets = false;
  for (var i = 0; i < lines.length; i++) {
    final line = lines[i];
    if (line.trim() == 'assets:') {
      inAssets = true;
      continue;
    }
    if (!inAssets) continue;
    if (line.isNotEmpty && !line.startsWith(' ') && !line.startsWith('#')) {
      break;
    }
    if (!line.contains(assetPath)) continue;
    final platforms = <String>[];
    for (var j = i + 1; j < lines.length; j++) {
      final next = lines[j];
      if (next.trim().startsWith('- path:') ||
          next.trim().startsWith('- assets/')) {
        break;
      }
      final name = RegExp(r'^ +- ([a-z]+)\s*$').firstMatch(next);
      if (name != null) platforms.add(name.group(1)!);
    }
    if (platforms.isEmpty) return true;
    return platforms.contains('web');
  }
  fail('pubspec.yaml has no asset entry for $assetPath');
}
