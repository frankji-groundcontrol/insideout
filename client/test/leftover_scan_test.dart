import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Live Dart under lib/ must not keep Ink & Seal tokens, chops, or seal assets.
void main() {
  test('live client dart has no celadon, vermilion, chops, or InkSeal', () {
    final files = Directory('lib')
        .listSync(recursive: true)
        .whereType<File>()
        .where((f) => f.path.endsWith('.dart'))
        .toList();
    expect(files, isNotEmpty);

    final forbidden = <String>[
      'InkSeal',
      'inkSealTheme',
      'SealChip',
      'SealMark',
      'SealTone',
      'assets/seals/',
      '0xFFC8402F',
      '0xFFE4EAE4',
      '0xFF161A17',
      'celadon',
      'vermilion',
    ];

    final hits = <String>[];
    for (final file in files) {
      final text = file.readAsStringSync();
      for (final needle in forbidden) {
        if (text.contains(needle)) {
          hits.add('${file.path}: $needle');
        }
      }
    }
    expect(
      hits,
      isEmpty,
      reason: 'Ink & Seal leftovers on live surfaces:\n${hits.join('\n')}',
    );
  });
}
