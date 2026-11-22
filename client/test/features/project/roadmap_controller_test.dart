import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_controller.dart';

import '_project_model_helpers.dart';

// STRC-05: the roadmap page's data lives in RoadmapController. The scripted
// API answers the reads, plays one status per stream call (then stays open)
// and hands the test the stream callbacks so it can play the server.
class _ScriptedApi extends ApiClient {
  _ScriptedApi({this.progressScript = const []})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final List<int> progressScript;
  final reads = <String>[];
  int progressCalls = 0;
  int presenceCalls = 0;
  void Function(RoadmapProgress)? onProgress;
  void Function(List<PresenceEntry>)? onPresence;
  void Function(String, String, double, double)? onCursor;
  final _open = Completer<void>();

  static RoadmapNode node(String id, {String? parentId, String? ownerId}) =>
      RoadmapNode.fromJson({
        'id': id,
        'projectId': 'p1',
        'parentId': parentId,
        'title': id,
        'description': '',
        'status': 'pending',
        'position': 0,
        'ownerId': ownerId,
        'canWrite': true,
      });

  static RoadmapProgress progress(int done) => RoadmapProgress(
    now: const [],
    needsDeadline: const [],
    next: const [],
    doneCount: done,
  );

  @override
  Future<ProjectDetail> getProject(String id) async =>
      projectDetail({'id': id, 'workspaceId': 'ws-1', 'canWritePlan': true});

  @override
  Future<List<WorkspaceMember>> listMembers(String workspaceId) async => [
    WorkspaceMember.fromJson({
      'userId': 'u-dana',
      'username': 'dana',
      'email': 'dana@test.local',
      'role': 'member',
      'joinedAt': '2026-01-01T00:00:00Z',
    }),
  ];

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async {
    reads.add('tree');
    return [node('n1')];
  }

  @override
  Future<List<RoadmapNode>> listRoadmapSubtree(
    String projectId,
    String rootId,
  ) async {
    reads.add('subtree $rootId');
    return [node('n2', parentId: 'n1'), node('n3', parentId: 'n2')];
  }

  @override
  Future<RoadmapProgress> roadmapProgress(String projectId) async =>
      progress(1);

  @override
  Future<RoadmapProgress> roadmapSubtreeProgress(
    String projectId,
    String rootId,
  ) async => progress(2);

  @override
  Future<RoadmapGlance> roadmapGlance(String nodeId) async =>
      RoadmapGlance.fromJson({
        'inCharge': {'userId': 'u-dana', 'name': 'dana'},
      });

  @override
  Future<RoadmapNodeRead> readRoadmapNode(String nodeId) async =>
      RoadmapNodeRead.fromJson({
        'node': {
          'id': nodeId,
          'projectId': 'p1',
          'parentId': 'n1',
          'title': nodeId,
          'description': '',
          'status': 'pending',
          'position': 0,
          'canWrite': true,
        },
        'ancestors': [
          {'id': 'n1', 'title': 'n1', 'ownerId': 'u-dana'},
        ],
        'projectId': 'p1',
      });

  @override
  Future<void> progressStream(
    String projectId,
    void Function(RoadmapProgress progress) onProgress, {
    Future<void>? stop,
  }) async {
    this.onProgress = onProgress;
    final call = progressCalls++;
    if (call < progressScript.length) {
      throw ApiException('down', status: progressScript[call]);
    }
    await _open.future;
  }

  @override
  Future<void> presenceStream(
    String projectId,
    String clientId,
    void Function(List<PresenceEntry> entries) onChange, {
    void Function(String sessionId, String name, double x, double y)? onCursor,
    Future<void>? stop,
  }) async {
    presenceCalls++;
    onPresence = onChange;
    this.onCursor = onCursor;
    await _open.future;
  }
}

PresenceEntry _entry(String sessionId) => PresenceEntry.fromJson({
  'sessionId': sessionId,
  'userId': 'u-$sessionId',
  'name': sessionId,
});

void main() {
  testWidgets('the whole tree reads the project, streams and reloads', (
    tester,
  ) async {
    final api = _ScriptedApi();
    final c = RoadmapController(api, projectId: 'p1');
    var notified = 0;
    c.addListener(() => notified++);
    await tester.pump();

    expect(api.reads, ['tree']);
    expect((await c.nodes).single.id, 'n1');
    expect((await c.progress).doneCount, 1);
    expect(c.read, isNull);
    expect(c.planWriter, isTrue);
    expect(api.progressCalls, 1);
    expect(api.presenceCalls, 1);

    api.onProgress!(_ScriptedApi.progress(5));
    expect(c.liveProgress?.doneCount, 5);
    expect(notified, greaterThan(0));

    await c.reload();
    expect(api.reads, ['tree', 'tree']);
    expect(c.liveProgress, isNull, reason: 'a reload shows the read reading');
    c.dispose();
  });

  testWidgets('presence keeps the cursors of the sessions still here', (
    tester,
  ) async {
    final api = _ScriptedApi();
    final c = RoadmapController(api, projectId: 'p1');
    await tester.pump();

    api.onPresence!([_entry('a'), _entry('b')]);
    api.onCursor!('a', 'a', 1, 2);
    api.onCursor!('b', 'b', 3, 4);
    expect(c.cursors.keys, unorderedEquals(['a', 'b']));

    api.onPresence!([_entry('a')]);
    expect(c.presence.map((e) => e.sessionId), ['a']);
    expect(c.cursors.keys, ['a']);
    c.dispose();
  });

  testWidgets('an opened subtree reads its own nodes and serves Mine', (
    tester,
  ) async {
    final api = _ScriptedApi();
    final c = RoadmapController(api, projectId: 'p1', rootId: 'n2');
    await tester.pump();
    await tester.pump();

    expect(api.reads, ['subtree n2']);
    expect((await c.progress).doneCount, 2);
    expect(api.progressCalls, 0, reason: 'the live reading is project-wide');
    expect(c.rootWritable, isTrue);
    expect(c.opened?.ancestors.single.id, 'n1');
    expect(c.names, {'u-dana': 'dana'}, reason: 'an ancestor has an owner');
    expect(c.glance?.inCharge?.userId, 'u-dana');

    final all = await c.nodes;
    // The caller owns the ancestor: the whole opened tree is theirs.
    expect(c.mine(all, 'u-dana'), same(all));
    // Someone else owns nothing here.
    expect(c.mine(all, 'u-other'), isEmpty);
    c.dispose();
  });

  testWidgets('a dropped stream retries after 500ms; dispose stops it', (
    tester,
  ) async {
    final api = _ScriptedApi(progressScript: [500, 500]);
    final c = RoadmapController(api, projectId: 'p1');
    await tester.pump();
    expect(api.progressCalls, 1);

    await tester.pump(const Duration(milliseconds: 500));
    await tester.pump();
    expect(api.progressCalls, 2);

    // Disposed mid back-off: no timer is left behind and no retry follows.
    c.dispose();
    await tester.pump(const Duration(seconds: 10));
    expect(api.progressCalls, 2);
  });

  testWidgets('a 403 ends the retrying for good', (tester) async {
    final api = _ScriptedApi(progressScript: [403]);
    final c = RoadmapController(api, projectId: 'p1');
    await tester.pump();
    await tester.pump(const Duration(seconds: 10));
    expect(api.progressCalls, 1);
    c.dispose();
  });
}
