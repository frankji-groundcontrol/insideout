import 'dart:collection';
import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models_roadmap.dart';

// rv5-COMP-02: server/internal/api/testdata/roadmap_node_full.json is a real
// GET /projects/{pid}/roadmap node with every optional field set, and
// roadmap_node_shape_test.go re-checks on every run that the server still
// answers in that shape. This test fails when RoadmapNode.fromJson leaves one
// of its keys (a rollup key included) unread, so a field the server adds
// reaches the model or a test says so.

/// A read-only JSON object that records every key read from it, nested
/// objects as "parent.key".
class _ReadRecorder extends MapBase<String, dynamic> {
  _ReadRecorder(this._m, this.read, [this._prefix = '']);

  final Map<String, dynamic> _m;
  final Set<String> read;
  final String _prefix;

  @override
  dynamic operator [](Object? key) {
    read.add('$_prefix$key');
    final v = _m[key];
    return v is Map<String, dynamic>
        ? _ReadRecorder(v, read, '$_prefix$key.')
        : v;
  }

  @override
  Iterable<String> get keys => _m.keys;

  @override
  void operator []=(String key, dynamic value) =>
      throw UnsupportedError('read-only');

  @override
  void clear() => throw UnsupportedError('read-only');

  @override
  dynamic remove(Object? key) => throw UnsupportedError('read-only');
}

Set<String> _allKeys(Map<String, dynamic> m, [String prefix = '']) => {
  for (final e in m.entries) ...[
    '$prefix${e.key}',
    if (e.value is Map<String, dynamic>)
      ..._allKeys(e.value as Map<String, dynamic>, '$prefix${e.key}.'),
  ],
};

/// Keys the model leaves unread on purpose, each with its reason.
const _unreadOnPurpose = {
  'createdAt':
      'the web shows a node by its creator and editor names, not times',
  'updatedAt':
      'the web shows a node by its creator and editor names, not times',
};

void main() {
  test('RoadmapNode.fromJson reads every key of a fully set node', () {
    final fixture =
        jsonDecode(
              File(
                '../server/internal/api/testdata/roadmap_node_full.json',
              ).readAsStringSync(),
            )
            as Map<String, dynamic>;
    final read = <String>{};
    final node = RoadmapNode.fromJson(_ReadRecorder(fixture, read));

    final unread = _allKeys(
      fixture,
    ).difference(read).difference(_unreadOnPurpose.keys.toSet());
    expect(
      unread,
      isEmpty,
      reason:
          'RoadmapNode.fromJson never reads $unread: read it, or add it '
          'to _unreadOnPurpose with the reason',
    );
    expect(
      _unreadOnPurpose.keys.where(read.contains),
      isEmpty,
      reason: 'a key in _unreadOnPurpose is read now: drop its entry',
    );
    expect(node.ownerId, fixture['ownerId']);
    expect(node.completionReason, 'Shipped by hand.');
    expect(node.canWrite, isTrue);
    expect(node.rollup.earliestDeadline, isNotNull);
  });
}
