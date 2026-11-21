import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_node_card.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';
import '_project_model_helpers.dart';

// rv5-PARC-06: the node edit dialog sends only the fields the person changed.
// The stored owner has left the workspace, so the picker shows none; a rename
// must not send that owner back (the server refuses a non-member owner).
class _Api extends ApiClient {
  _Api()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  Map<String, Object?>? saved;
  var calls = 0;

  @override
  Future<ProjectDetail> getProject(String id) async =>
      projectDetail({'id': id, 'workspaceId': 'ws-1'});

  @override
  Future<List<WorkspaceMember>> listMembers(String id) async => [
    WorkspaceMember(
      userId: 'u-dana',
      username: 'dana',
      email: 'dana@test.local',
      role: 'member',
    ),
  ];

  @override
  Future<RoadmapNode> updateRoadmapNode(
    String nodeId, {
    String? title,
    String? description,
    String? status,
    String? deadline,
    String? acceptance,
    String? ownerId,
    String? completionReason,
  }) async {
    calls++;
    saved = {
      'title': title,
      'description': description,
      'status': status,
      'deadline': deadline,
      'acceptance': acceptance,
      'ownerId': ownerId,
      'completionReason': completionReason,
    };
    return node;
  }

  static final node = RoadmapNode(
    canWrite: true,
    id: 'n1',
    projectId: 'p1',
    title: 'Wire the flow',
    description: 'notes',
    status: 'pending',
    position: 0,
    deadline: '2026-12-01T12:00:00Z',
    acceptance: 'healthz answers 200',
    ownerId: 'u-gone',
  );
}

Future<void> _openEdit(WidgetTester tester, _Api api) async {
  final session = Session()
    ..ready = true
    ..api = api;
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'en-US'),
        ),
      ],
      child: MaterialApp(
        home: Scaffold(
          body: RoadmapNodeCard(
            planWriter: true,
            projectId: 'p1',
            node: _Api.node,
            onReload: () async {},
            descendants: 0,
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  await tester.tap(find.byType(PopupMenuButton<String>));
  await tester.pumpAndSettle();
  await tester.tap(find.text('Edit').last);
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('a rename sends the title alone, not the departed owner', (
    tester,
  ) async {
    final api = _Api();
    await _openEdit(tester, api);
    await tester.enterText(find.byType(TextField).first, 'Wire the login');
    await tester.pump();
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(api.saved, {
      'title': 'Wire the login',
      'description': null,
      'status': null,
      'deadline': null,
      'acceptance': null,
      'ownerId': null,
      'completionReason': null,
    });
  });

  testWidgets('a save with no change sends nothing', (tester) async {
    final api = _Api();
    await _openEdit(tester, api);
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
    expect(api.calls, 0);
  });
}
