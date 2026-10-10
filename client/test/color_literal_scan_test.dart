import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Cobalt is the only colour authority: live Dart under lib/ carries no
/// `Color(0x` hex literal outside the two theme token files, so hardcoded
/// paint cannot reappear at the seams the surfaces share.
void main() {
  test('no Color hex literal outside the cobalt token layer', () {
    final files = Directory('lib')
        .listSync(recursive: true)
        .whereType<File>()
        .where((f) => f.path.endsWith('.dart'))
        .where(
          (f) =>
              f.path != 'lib/theme/cobalt.dart' &&
              f.path != 'lib/theme/cobalt_theme.dart',
        )
        .toList();
    expect(files, isNotEmpty);

    final hits = <String>[
      for (final file in files)
        for (final (i, line) in file.readAsLinesSync().indexed)
          if (line.contains('Color(0x'))
            '${file.path}:${i + 1}: ${line.trim()}',
    ];
    expect(
      hits,
      isEmpty,
      reason: 'Hardcoded paint outside the token layer:\n${hits.join('\n')}',
    );
  });
}
