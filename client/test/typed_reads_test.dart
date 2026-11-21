import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';

import '_expand_fixture_helpers.dart';

// rv5-STRC-08: GET /projects/{id} and GET /roadmap/{id}/evidence parse into
// typed models. The bodies are real responses of a running server
// (server/internal/api/testdata/), re-checked by the Go test
// TestProjectAndEvidenceReadsKeepTheirCapturedShapes on every run; this feeds
// them to the real client.
void main() {
  test(
    'the project read parses into the project, its updates and cursor',
    () async {
      final raw = expandFixture('project_get.json');
      final sent = jsonDecode(raw) as Map<String, dynamic>;
      final server = ServerFixtures({'GET /api/v1/projects/p1': (200, raw)});

      final got = await clientOver(server).getProject('p1');

      expect(got.project.id, sent['id']);
      expect(got.project.workspaceId, sent['workspaceId']);
      expect(got.project.title, sent['title']);
      expect(got.project.ownerId, sent['ownerId']);
      expect(got.project.canWritePlan, sent['canWritePlan']);
      expect(got.project.prdId, isNull);
      expect(got.nextCursor, isNull);
      final updates = sent['updates'] as List;
      expect(updates, isNotEmpty);
      expect(
        [for (final u in got.updates) u.content],
        [for (final u in updates) (u as Map)['content']],
      );
    },
  );

  test('the evidence read parses every row', () async {
    final raw = expandFixture('roadmap_evidence_list.json');
    final sent = (jsonDecode(raw) as List).cast<Map<String, dynamic>>();
    final server = ServerFixtures({
      'GET /api/v1/roadmap/n1/evidence': (200, raw),
    });

    final got = await clientOver(server).listNodeEvidence('n1');

    expect(sent, isNotEmpty);
    expect([for (final e in got) e.id], [for (final e in sent) e['id']]);
    expect([for (final e in got) e.kind], [for (final e in sent) e['kind']]);
    expect(
      [for (final e in got) e.detail],
      [for (final e in sent) e['detail']],
    );
    expect(
      [for (final e in got) e.observedAt],
      [for (final e in sent) e['observedAt']],
    );
  });

  // The machine-appended kinds carry no observedAt (omitempty).
  test('an evidence row without observedAt parses', () async {
    final server = ServerFixtures({
      'GET /api/v1/roadmap/n1/evidence': (
        200,
        '[{"id":"e1","nodeId":"n1","kind":"implementation","detail":"merged",'
            '"sourceUrl":"https://example.test/pr/1","createdAt":"2026-11-01T00:00:00Z"}]',
      ),
    });

    final got = await clientOver(server).listNodeEvidence('n1');

    expect(got.single.observedAt, isNull);
    expect(got.single.sourceUrl, 'https://example.test/pr/1');
  });
}
