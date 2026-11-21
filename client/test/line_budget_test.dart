import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// One responsibility per file: past the shared threshold a file is usually
/// doing more than one thing. The threshold lives in config/code-budget.json,
/// which the server guard (server/internal/linebudget) reads too, so the two
/// limits cannot drift. A file that crosses it fails here until it is split
/// by responsibility or listed below with a dated modularity issue.
void main() {
  test(
    'every dart file under lib/ and test/ stays within the shared file budget',
    () {
      final config =
          jsonDecode(File('../config/code-budget.json').readAsStringSync())
              as Map<String, dynamic>;
      final budget = config['fileLines'] as int;
      expect(budget, greaterThan(0));

      final files = [
        for (final dir in ['lib', 'test'])
          ...Directory(dir)
              .listSync(recursive: true)
              .whereType<File>()
              .where((f) => f.path.endsWith('.dart')),
      ];
      expect(files, isNotEmpty);

      // Files allowed over the budget, path to current line count, each with
      // an open dated modularity issue in docs/issues/. A listed file may not
      // grow. None today.
      const overBudgetAllowed = <String, int>{};

      final over = <String>[];
      for (final file in files) {
        final lines = file.readAsLinesSync().length;
        final allowed = overBudgetAllowed[file.path];
        if (lines > (allowed ?? budget)) {
          over.add("'${file.path}': $lines,");
        }
      }
      expect(
        over,
        isEmpty,
        reason:
            'Files over the $budget-line budget (or past their listed count); '
            'split them, or paste into overBudgetAllowed with an issue:\n'
            '${over.join('\n')}',
      );
    },
  );
}
