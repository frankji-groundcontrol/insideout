import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// The dock clearance (ProductDock.clearance) may be spelled once, in the
/// dock. Pages must not keep private copies of the sum or literal 140s: a
/// dock resize once desynchronized twelve hand-maintained guesses at it.
void main() {
  // landing/assembly_diagram.dart staggers its animation in milliseconds;
  // its 140 is a delay, not a dock clearance.
  const msStaggerFile = 'landing/assembly_diagram.dart';
  final standalone140 = RegExp(r'\b140\b');
  final clearanceCopy = RegExp(
    r'ProductDock\.height\s*\+\s*Cobalt\.space10\s*\+\s*Cobalt\.space4',
  );

  test('feature pages keep no private copy of the dock clearance', () {
    final files = Directory('lib/features')
        .listSync(recursive: true)
        .whereType<File>()
        .where((f) => f.path.endsWith('.dart'))
        .toList();
    expect(files, isNotEmpty);

    final hits = <String>[
      for (final file in files)
        for (final (i, line) in file.readAsLinesSync().indexed)
          if (!file.path.endsWith(msStaggerFile) &&
              standalone140.hasMatch(line))
            '${file.path}:${i + 1}: ${line.trim()}'
          else if (clearanceCopy.hasMatch(line))
            '${file.path}:${i + 1}: ${line.trim()}',
    ];
    expect(
      hits,
      isEmpty,
      reason: 'Dock clearance spelled outside ProductDock.clearance:\n'
          '${hits.join('\n')}',
    );
  });
}
