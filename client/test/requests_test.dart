import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/requests.dart';

void main() {
  test('workspace update and delete match Nuxt', () {
    final patch = patchWorkspace('ws1', title: 'T', description: 'D', coverUrl: null);
    expect(patch.method, 'PATCH');
    expect(patch.path, '/workspaces/ws1');
    expect(patch.body, {'title': 'T', 'description': 'D', 'coverUrl': null});
    final del = deleteWorkspace('ws1');
    expect(del.method, 'DELETE');
    expect(del.path, '/workspaces/ws1');
    expect(del.body, isNull);
  });

  test('member role and remove match Nuxt', () {
    final role = patchMemberRole('ws1', 'u2', 'admin');
    expect(role.method, 'PATCH');
    expect(role.path, '/workspaces/ws1/members/u2');
    expect(role.body, {'role': 'admin'});
    final remove = deleteMember('ws1', 'u2');
    expect(remove.method, 'DELETE');
    expect(remove.path, '/workspaces/ws1/members/u2');
  });

  test('idea patch and delete match Nuxt', () {
    final patch = patchIdea('i1', title: 'Idea', content: 'Body');
    expect(patch.method, 'PATCH');
    expect(patch.path, '/ideas/i1');
    expect(patch.body, {'title': 'Idea', 'content': 'Body'});
    expect(deleteIdea('i1').method, 'DELETE');
    expect(deleteIdea('i1').path, '/ideas/i1');
  });

  test('project patch/delete and update patch/delete match Nuxt', () {
    final patch = patchProject('p1', title: 'P', description: 'd', status: 'active', ownerId: 'u1');
    expect(patch.method, 'PATCH');
    expect(patch.path, '/projects/p1');
    expect(patch.body, {'title': 'P', 'description': 'd', 'status': 'active', 'ownerId': 'u1'});
    expect(deleteProject('p1').path, '/projects/p1');
    expect(deleteProject('p1').method, 'DELETE');
    final edit = patchProjectUpdate('up1', 'revised');
    expect(edit.method, 'PATCH');
    expect(edit.path, '/updates/up1');
    expect(edit.body, {'content': 'revised'});
    expect(deleteProjectUpdate('up1').method, 'DELETE');
    expect(deleteProjectUpdate('up1').path, '/updates/up1');
  });

  test('roadmap update move delete expand match Nuxt', () {
    final patch = patchRoadmapNode('n1', title: 'N', status: 'done');
    expect(patch.method, 'PATCH');
    expect(patch.path, '/roadmap/n1');
    expect(patch.body, {'title': 'N', 'status': 'done'});
    final move = moveRoadmapNode('n1', parentId: null, position: 2);
    expect(move.method, 'POST');
    expect(move.path, '/roadmap/n1/move');
    expect(move.body, {'parentId': null, 'position': 2});
    expect(deleteRoadmapNode('n1').method, 'DELETE');
    expect(deleteRoadmapNode('n1').path, '/roadmap/n1');
    expect(expandRoadmapNode('n1').method, 'POST');
    expect(expandRoadmapNode('n1').path, '/roadmap/n1/expand');
    expect(expandRoadmapNode('n1').body, isNull);
  });

  test('GET idea and conversation match Nuxt', () {
    expect(getIdea('i1').method, 'GET');
    expect(getIdea('i1').path, '/ideas/i1');
    expect(getIdea('i1').body, isNull);
    expect(getConversation('c1').method, 'GET');
    expect(getConversation('c1').path, '/conversations/c1');
  });

  test('revision snapshot omits body unless a note is set', () {
    final bare = postPrdRevision('prd1');
    expect(bare.method, 'POST');
    expect(bare.path, '/prds/prd1/revisions');
    expect(bare.body, isNull);
    expect(postPrdRevision('prd1', note: 'before review').body, {'note': 'before review'});
  });

  test('roadmap create sends parentId only for a child', () {
    final root = postRoadmapNode('p1', 'Ship');
    expect(root.method, 'POST');
    expect(root.path, '/projects/p1/roadmap');
    expect(root.body, {'title': 'Ship'});
    expect(postRoadmapNode('p1', 'Task', parentId: 'n1').body, {'title': 'Task', 'parentId': 'n1'});
  });

  test('PRD status and build-from-PRD first call vs confirm', () {
    final status = postPrdStatus('prd1', 'approved');
    expect(status.method, 'POST');
    expect(status.path, '/prds/prd1/status');
    expect(status.body, {'status': 'approved'});
    final first = postPrdBuild('prd1');
    expect(first.method, 'POST');
    expect(first.path, '/prds/prd1/build');
    expect(first.body, isNull);
    final confirm = postPrdBuild('prd1', expectedCount: 4);
    expect(confirm.body, {'expectedCount': 4});
  });
}
