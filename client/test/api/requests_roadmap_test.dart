import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/requests.dart';

// rv5-PARC-12: every roadmap and GitHub request the web sends goes through an
// ApiCall builder, so its method, path and body are asserted here.
void main() {
  void expectCall(ApiCall c, String method, String path, [Object? body]) {
    expect(c.method, method);
    expect(c.path, path);
    expect(c.body, body);
  }

  test('roadmap reads: tree, subtree, node, assignments, evidence', () {
    expectCall(getRoadmap('p1'), 'GET', '/projects/p1/roadmap');
    expectCall(
      getRoadmap('p1', rootId: 'n 1'),
      'GET',
      '/projects/p1/roadmap?root=n+1',
    );
    expectCall(getRoadmapNode('n1'), 'GET', '/roadmap/n1');
    expectCall(getNodeAssignments('n1'), 'GET', '/roadmap/n1/assignments');
    expectCall(getNodeEvidence('n1'), 'GET', '/roadmap/n1/evidence');
  });

  test('roadmap progress over the project or one subtree', () {
    expectCall(
      getRoadmapProgress('p1'),
      'GET',
      '/projects/p1/roadmap/progress',
    );
    expectCall(
      getRoadmapProgress('p1', rootId: 'n1'),
      'GET',
      '/projects/p1/roadmap/progress?root=n1',
    );
  });

  test('validation: an observation, then the confirm', () {
    expectCall(
      postNodeEvidence(
        'n1',
        detail: 'seen',
        source: 'https://example.test',
        observedAt: DateTime.utc(2026, 11, 2, 15),
      ),
      'POST',
      '/roadmap/n1/evidence',
      {
        'detail': 'seen',
        'sourceUrl': 'https://example.test',
        'observedAt': '2026-11-02',
      },
    );
    expectCall(postValidate('n1', 'ok'), 'POST', '/roadmap/n1/validate', {
      'reason': 'ok',
    });
  });

  test('cursor names the client it speaks for', () {
    expectCall(
      postCursor('p1', 12, 34, clientId: 'c1'),
      'POST',
      '/projects/p1/cursor?client=c1',
      {'x': 12.0, 'y': 34.0},
    );
  });

  test('repository, sync, updates and the proposal decision', () {
    expectCall(putProjectRepo('p1', 'o/r'), 'PUT', '/projects/p1/repo', {
      'repoUrl': 'o/r',
    });
    expectCall(postSyncGithub('p1'), 'POST', '/projects/p1/sync-github');
    expectCall(
      postProjectUpdate('p1', 'note', 'hi'),
      'POST',
      '/projects/p1/updates',
      {'kind': 'note', 'content': 'hi'},
    );
    expectCall(
      postProposalDecision('u1', accept: false, reason: 'no', apply: true),
      'POST',
      '/agent/proposals/u1/decision',
      {'decision': 'rejected', 'reason': 'no', 'apply': true},
    );
  });
}
