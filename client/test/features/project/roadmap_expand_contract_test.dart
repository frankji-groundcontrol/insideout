import 'package:flutter_test/flutter_test.dart';

import '_expand_fixture_helpers.dart';

// rv4-FULL-01: "Break down with AI" on the web always failed. The server's
// POST /roadmap/{id}/expand answers 200 {"children":[{title, description}]}
// and writes nothing; the client cast that map to a List of roadmap nodes
// (`(d as List)`), and no client code ever called the confirm route. Nothing
// caught it because the request tests pinned the path only. This feeds the
// real client the response a running server really sent (testdata/, shared
// with the Go test that re-checks the server's shape on every run).
void main() {
  test('the draft the server sends parses into subtasks', () async {
    final server = ServerFixtures.captured('n1');
    final draft = await clientOver(server).expandRoadmapNode('n1');

    final sent = capturedDraftChildren();
    expect(sent, isNotEmpty);
    expect(
      [for (final c in draft) c.title],
      [for (final c in sent) c['title']],
    );
    expect(
      [for (final c in draft) c.description],
      [for (final c in sent) c['description']],
    );
    // Asking for a draft is one request: nothing is confirmed on the way.
    expect(server.calls, ['POST /api/v1/roadmap/n1/expand']);
  });

  // The server drops an empty description (`json:"description,omitempty"`).
  test('a drafted subtask without a description keeps its title', () async {
    final server = ServerFixtures({
      'POST /api/v1/roadmap/n1/expand': (
        200,
        '{"children":[{"title":"Write the tests"}]}',
      ),
    });
    final draft = await clientOver(server).expandRoadmapNode('n1');

    expect(draft, hasLength(1));
    expect(draft.single.title, 'Write the tests');
    expect(draft.single.description, '');
  });
}
