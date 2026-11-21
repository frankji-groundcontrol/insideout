import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// CQ-01 (auth door scope): spacing is the named Cobalt scale, so the auth
/// door carries no bare numeric gaps or paddings. A raw number here keeps
/// the old rhythm when the scale retunes and no test fails — the scan makes
/// one fail. Modeled on color_literal_scan_test.dart, scoped to this file
/// because the repo-wide sweep is being tokenized file-by-file.
void main() {
  test(
    'the auth door spells its spacing in Cobalt tokens, not bare numbers',
    () {
      const path = 'lib/features/auth/auth_door.dart';
      final file = File(path);
      expect(file.existsSync(), isTrue);

      final rawGap = RegExp(r'SizedBox\((height|width):\s*[0-9]');
      // `EdgeInsets.only(bottom: 80)` and `EdgeInsets.fromLTRB(20, …)` both
      // put a digit where a token belongs.
      final rawPadding = RegExp(
        r'EdgeInsets\.[a-zA-Z]+\((\s*[0-9]|[^)]*:\s*[0-9])',
      );

      final hits = <String>[
        for (final (i, line) in file.readAsLinesSync().indexed)
          if (rawGap.hasMatch(line) || rawPadding.hasMatch(line))
            '$path:${i + 1}: ${line.trim()}',
      ];
      expect(
        hits,
        isEmpty,
        reason: 'Raw numeric spacing in the auth door:\n${hits.join('\n')}',
      );
    },
  );
}
