import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/project_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// FEAT-03 behavioral coverage: agent proposals render a decision loop
// (Accept/Reject against the existing decision endpoint) instead of the
// generic edit/delete menu, which 403s for non-admins and never decides
// anything anyway.
class _FakeApi extends ApiClient {
  _FakeApi(this.updates, {this.onDecide, this.nodes = const []}) : super(
            baseUrl: 'http://127.0.0.1:9/api/v1',
            readAccess: () async => null,
            readRefresh: () async => null,
            writeTokens: (a, r) async {},
            clearTokens: () async {},
          );

  final List<Map<String, dynamic>> updates;
  final List<RoadmapNode> nodes;
  void Function(String id, bool accept)? onDecide;
  final List<({String projectId, String title, String? parentId})> created = [];

  @override
  Future<Map<String, dynamic>> getProject(String id) async => {
        'id': id,
        'workspaceId': 'ws1',
        'title': 'Evidence Page Test',
        'description': '',
        'status': 'active',
        'repoUrl': '',
        'ownerId': 'u1',
        'updates': updates,
      };

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => nodes;

  @override
  Future<RoadmapNode> createRoadmapNode(String projectId, String title, {String? parentId}) async {
    created.add((projectId: projectId, title: title, parentId: parentId));
    return RoadmapNode(
      id: 'node-${created.length}',
      projectId: projectId,
      title: title,
      description: '',
      status: 'pending',
      position: created.length,
      parentId: parentId,
    );
  }

  @override
  Future<Map<String, dynamic>> decideProposal(String updateId, {required bool accept, String reason = '', bool apply = true}) async {
    onDecide?.call(updateId, accept);
    return {'decision': accept ? 'accepted' : 'rejected'};
  }
}

Map<String, dynamic> _update(String id, String kind, {List<Map<String, dynamic>>? items}) => {
      'id': id,
      'kind': kind,
      'content': kind == 'agent_proposal' ? 'Propose adding login throttling' : 'shipped the flow',
      'createdAt': '2026-10-01T00:00:00Z',
      'authorId': 'agent',
      if (items != null) 'items': items,
    };

Future<void> _pump(WidgetTester tester, _FakeApi api) async {
  final session = Session();
  session.ready = true;
  session.api = api;
  await tester.binding.setSurfaceSize(const Size(1400, 900));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US')),
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: const MaterialApp(home: ProjectPage(id: 'p1')),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('agent proposal gets Accept/Reject; notes keep edit/delete', (tester) async {
    final decided = <String>[];
    final api = _FakeApi([_update('up1', 'agent_proposal'), _update('up2', 'note')], onDecide: (id, accept) => decided.add('$id:${accept ? 'y' : 'n'}'));
    await _pump(tester, api);

    expect(find.text('Propose adding login throttling'), findsOneWidget);
    expect(find.text('Accept'), findsOneWidget);
    expect(find.text('Reject'), findsOneWidget);

    // The proposal carries no edit/delete menu (it would 403 and decide nothing).
    expect(find.byIcon(Icons.more_vert), findsOneWidget, reason: 'exactly one menu — the note\'s, not the proposal\'s');

    await tester.tap(find.text('Accept'));
    await tester.pumpAndSettle();
    expect(decided, ['up1:y']);
  });

  testWidgets('rejecting a proposal posts the rejected decision', (tester) async {
    final decided = <String>[];
    final api = _FakeApi([_update('up1', 'agent_proposal')], onDecide: (id, accept) => decided.add('$id:${accept ? 'y' : 'n'}'));
    await _pump(tester, api);

    await tester.tap(find.text('Reject'));
    await tester.pumpAndSettle();
    expect(decided, ['up1:n']);
  });

  testWidgets('accepting a proposal with one item adds that roadmap node', (tester) async {
    final api = _FakeApi([
      _update('up1', 'agent_proposal', items: [
        {'action': 'add_node', 'title': 'Login throttling', 'parentHint': ''},
      ]),
    ]);
    await _pump(tester, api);

    await tester.tap(find.text('Accept'));
    await tester.pumpAndSettle();

    expect(api.created, [(projectId: 'p1', title: 'Login throttling', parentId: null)]);
  });
}
