import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Wave 7.1: the web bundle must not ship the two CJK variable fonts.
/// Native platforms still list them. Web loads the faces from the CDN.
void main() {
  test('declared web assets omit the two CJK variable fonts', () {
    final pubspec = File('pubspec.yaml').readAsStringSync();
    expect(pubspec.contains('assets/fonts/OFL.txt'), isTrue);
    expect(_bundledOnWeb(pubspec, 'assets/fonts/NotoSansSC-Var.ttf'), isFalse);
    expect(_bundledOnWeb(pubspec, 'assets/fonts/NotoSerifSC-Var.ttf'), isFalse);
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
