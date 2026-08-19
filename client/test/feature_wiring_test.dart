import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('existing screens invoke every new mutation and extra SSE handlers', () {
    String read(String path) => File(path).readAsStringSync();
    final settings = read('lib/features/workspace/settings_page.dart');
    final workspace = read('lib/features/workspace/workspace_page.dart');
    final ideas = read('lib/features/workspace/ideas_page.dart');
    final project = read('lib/features/project/project_page.dart');
    final roadmap = read('lib/features/project/roadmap_page.dart');
    final prd = read('lib/features/prd/prd_page.dart') + read('lib/features/prd/coach_panel.dart');
    final client = read('lib/api/client.dart');

    expect(settings.contains('updateWorkspace'), isTrue);
    expect(settings.contains('deleteWorkspace'), isTrue);
    expect(settings.contains('updateMemberRole'), isTrue);
    expect(settings.contains('removeMember'), isTrue);
    expect(workspace.contains('updateWorkspace'), isTrue);
    expect(workspace.contains('deleteWorkspace'), isTrue);
    expect(ideas.contains('updateIdea'), isTrue);
    expect(ideas.contains('dropIdea'), isTrue);
    expect(project.contains('updateProject'), isTrue);
    expect(project.contains('deleteProject'), isTrue);
    expect(project.contains('editUpdate'), isTrue);
    expect(project.contains('removeUpdate'), isTrue);
    expect(roadmap.contains('updateRoadmapNode'), isTrue);
    expect(roadmap.contains('moveRoadmapNode'), isTrue);
    expect(roadmap.contains('deleteRoadmapNode'), isTrue);
    expect(roadmap.contains('expandRoadmapNode'), isTrue);
    expect(roadmap.contains("parentId: n.id"), isTrue);
    expect(prd.contains('updatePrdStatus'), isTrue);
    expect(prd.contains('buildFromPrd'), isTrue);
    expect(prd.contains('expectedCount'), isTrue);
    expect(prd.contains('onPrdUpdated'), isTrue);
    expect(prd.contains('onStageChanged'), isTrue);
    expect(prd.contains('onFactRecorded'), isTrue);
    expect(prd.contains('coach.noConversation'), isTrue);
    expect(prd.contains('suggestions('), isTrue);
    expect(prd.contains('isCoachBackoff'), isTrue);
    expect(client.contains('applyCoachFrame'), isTrue);
    expect(client.contains('req.postPrdBuild'), isTrue);
    expect(client.contains('req.getIdea'), isTrue);
    expect(client.contains('req.getConversation'), isTrue);
    expect(client.contains('req.postPrdRevision'), isTrue);
    expect(workspace.contains('inviteCode'), isTrue);
  });
}
