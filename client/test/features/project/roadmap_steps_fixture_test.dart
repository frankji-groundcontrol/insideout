import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_step_flow.dart';

// server/internal/roadmaptime/testdata/steps_cases.json is the shared step
// fixture: GET /roadmap/steps (roadmaptime.Steps) and the Canvas
// (roadmap_step_flow.dart) must order every case the same way.

List<RoadmapNode> _nodes(List<dynamic> raw) => [
  for (final n in raw.cast<Map<String, dynamic>>())
    RoadmapNode(
      id: n['id'] as String,
      projectId: 'p',
      parentId: (n['parentId'] as String).isEmpty
          ? null
          : n['parentId'] as String,
      title: n['id'] as String,
      description: '',
      status: 'pending',
      position: n['position'] as int,
      dependsOn: (n['dependsOn'] as List).cast<String>(),
    ),
];

List<RoadmapNode> _steps(List<RoadmapNode> all, String root) {
  if (root.isEmpty) return roadmapSteps(all);
  final kids = all.where((n) => n.parentId == root).toList()
    ..sort((a, b) => a.position.compareTo(b.position));
  return kids.isEmpty ? all.where((n) => n.id == root).toList() : kids;
}

void main() {
  final file = File('../server/internal/roadmaptime/testdata/steps_cases.json');
  final cases = (jsonDecode(file.readAsStringSync())['cases'] as List)
      .cast<Map<String, dynamic>>();

  for (final c in cases) {
    test('steps match the server: ${c['name']}', () {
      final all = _nodes(c['nodes'] as List);
      final flow = RoadmapStepFlow.of(_steps(all, c['root'] as String), all);
      // The server lists `after` in step order; the arrows' order is the
      // painter's business, so compare in step order.
      final rank = {
        for (final (i, s) in flow.columns.expand((c) => c).indexed) s.id: i,
      };
      final got = <Map<String, dynamic>>[];
      for (var col = 0; col < flow.columns.length; col++) {
        for (final s in flow.columns[col]) {
          got.add({
            'id': s.id,
            'stage': col,
            'after': [
              for (final e in flow.edges)
                if (e.to == s.id) e.from,
            ]..sort((a, b) => rank[a]!.compareTo(rank[b]!)),
            'loop': flow.loops.firstWhere(
              (l) => l.contains(s.id),
              orElse: () => const [],
            ),
          });
        }
      }
      final want = (c['expect'] as List).cast<Map<String, dynamic>>();
      expect(jsonEncode(got), jsonEncode(want));
    });
  }
}
