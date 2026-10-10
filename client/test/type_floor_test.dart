import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// DESIGN.md: Meta (600, 12) is the type floor — chips, eyebrows and dock
/// labels never drop below 12px. The floor is scanned across live Dart, so a
/// sub-12 literal cannot ship silently.
void main() {
  test('no fontSize literal below 12 ships in live client dart', () {
    final files = Directory('lib')
        .listSync(recursive: true)
        .whereType<File>()
        .where((f) => f.path.endsWith('.dart'))
        .toList();
    expect(files, isNotEmpty);

    final pattern = RegExp(r'fontSize:\s*([0-9]+(?:\.[0-9]+)?)');
    final hits = <String>[];
    for (final file in files) {
      for (final (i, line) in file.readAsLinesSync().indexed) {
        final match = pattern.firstMatch(line);
        if (match != null && double.parse(match.group(1)!) < 12) {
          hits.add('${file.path}:${i + 1}: ${line.trim()}');
        }
      }
    }
    expect(
      hits,
      isEmpty,
      reason:
          'Sub-12px fontSize literals on live surfaces:\n${hits.join('\n')}',
    );
  });
}
