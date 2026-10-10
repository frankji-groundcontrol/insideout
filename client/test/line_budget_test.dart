import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// AGENTS.md keeps implementation files under ~350 lines: past that, a file
/// is doing more than one thing. The 2026-10-10 sweep brought every client
/// file back under the budget, but nothing guarded it — coach_panel.dart
/// drifted to 354 lines within hours. This scan is the guard: a file that
/// crosses the budget fails here until it is split or listed below with a
/// dated modularity issue.
void main() {
  test(
    'every dart file under lib/ and test/ stays within the 350-line budget',
    () {
      final files = [
        for (final dir in ['lib', 'test'])
          ...Directory(dir)
              .listSync(recursive: true)
              .whereType<File>()
              .where((f) => f.path.endsWith('.dart')),
      ];
      expect(files, isNotEmpty);

      const budget = 350;
      // Files allowed over the budget, each with an open dated modularity
      // issue in docs/issues/. None today.
      const overBudgetAllowed = <String>[];

      final over = <String>[];
      for (final file in files) {
        final lines = file.readAsLinesSync().length;
        if (lines > budget && !overBudgetAllowed.contains(file.path)) {
          over.add('${file.path}: $lines lines');
        }
      }
      expect(
        over,
        isEmpty,
        reason: 'Files over the $budget-line budget:\n${over.join('\n')}',
      );
    },
  );
}
