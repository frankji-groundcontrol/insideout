import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Live Dart under lib/ must not keep Ink & Seal tokens, chops, or seal assets,
/// and no test may cite the deleted Nuxt app as the contract it pins.
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

  // The Go server is the contract now; a test that still calls the Nuxt app
  // its oracle points the next reader at code that no longer exists. Name the
  // route, status or string the test asserts instead.
  test('no client test names the deleted Nuxt app as its contract', () {
    final files = Directory('test')
        .listSync(recursive: true)
        .whereType<File>()
        .where(
          // This file spells the needle out, so it is the one not scanned.
          (f) =>
              f.path.endsWith('.dart') &&
              !f.path.endsWith('leftover_scan_test.dart'),
        )
        .toList();
    expect(files, isNotEmpty);

    final hits = <String>[
      for (final file in files)
        for (final (i, line) in file.readAsLinesSync().indexed)
          if (line.toLowerCase().contains('nuxt'))
            '${file.path}:${i + 1}: ${line.trim()}',
    ];
    expect(
      hits,
      isEmpty,
      reason: 'Tests citing the deleted Nuxt app:\n${hits.join('\n')}',
    );
  });
}
