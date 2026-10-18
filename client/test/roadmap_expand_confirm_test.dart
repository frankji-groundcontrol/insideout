import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/requests.dart';

import '_expand_fixture_helpers.dart';

// rv4-FULL-01: the second step of the AI breakdown. The web client never
// called POST /roadmap/{id}/expand/confirm, the only route that writes the
// drafted subtasks; this pins the call it makes and how it reads the 201 list
// the server answers with (testdata/roadmap_expand_confirm.json, captured
// from a running server).
void main() {
  test('the confirm request carries the drafted subtasks', () {
    final call = confirmExpandRoadmapNode('n1', [
      {'title': 'Write the tests', 'description': 'Red first'},
      {'title': 'Ship it', 'description': ''},
    ]);

    expect(call.method, 'POST');
    expect(call.path, '/roadmap/n1/expand/confirm');
    expect(call.body, {
      'children': [
        {'title': 'Write the tests', 'description': 'Red first'},
        {'title': 'Ship it', 'description': ''},
      ],
    });
  });

  test(
    'confirming sends the draft as drafted and parses the 201 list',
    () async {
      final server = ServerFixtures.captured('n1');
      final api = clientOver(server);

      final draft = await api.expandRoadmapNode('n1');
      final created = await api.confirmExpandRoadmapNode('n1', draft);

      expect(server.calls, [
        'POST /api/v1/roadmap/n1/expand',
        'POST /api/v1/roadmap/n1/expand/confirm',
      ]);
      expect(server.requests.last.data, {'children': capturedDraftChildren()});

      // One node per drafted subtask, all under the node that was expanded.
      final sent = capturedDraftChildren();
      expect(created, hasLength(sent.length));
      expect(
        [for (final n in created) n.title],
        [for (final c in sent) c['title']],
      );
      expect(created.map((n) => n.status).toSet(), {'pending'});
      expect(created.map((n) => n.parentId).toSet(), hasLength(1));
      expect(created.map((n) => n.id).toSet(), hasLength(sent.length));
      expect(
        [for (final n in created) n.position],
        [for (var i = 0; i < sent.length; i++) i],
      );
    },
  );
}
