import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/l10n/en.dart';

/// A locale key nothing reads is a feature that was dropped, or never built,
/// and it still costs translation upkeep and tells the next reader the feature
/// exists. The Cobalt restyle dropped the carried-item list, the section diff
/// and the decision note from the version card this way; the only trace was
/// their strings. A key counts as read when its literal appears in lib/ outside
/// the tables, or when it sits under a prefix built at runtime
/// (`'prd.sections.$key'`).
void main() {
  test('every locale key is read somewhere in lib/', () {
    final sources = Directory('lib')
        .listSync(recursive: true)
        .whereType<File>()
        .where((f) => f.path.endsWith('.dart') && !f.path.contains('/l10n/'))
        .map((f) => f.readAsStringSync())
        .join('\n');
    final prefixes = RegExp(
      r'''['"]([A-Za-z][\w.]*\.)\$''',
    ).allMatches(sources).map((m) => m[1]!).toSet();

    final unread = [
      for (final key in enUS.keys)
        if (!sources.contains("'$key'") &&
            !sources.contains('"$key"') &&
            !prefixes.any(key.startsWith))
          key,
    ];

    expect(
      unread,
      isEmpty,
      reason:
          'Use these, or delete them from every en_*.dart and zh_*.dart '
          'table together:\n${unread.join('\n')}',
    );
  });
}
