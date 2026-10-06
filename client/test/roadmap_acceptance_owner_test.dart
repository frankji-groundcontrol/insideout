import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_node_card.dart';
import 'package:insideout/features/project/roadmap_progress_view.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

class _MembersApi extends ApiClient {
  _MembersApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  Map<String, dynamic>? saved;

  @override
  Future<Map<String, dynamic>> getProject(String id) async => {
    'id': id,
    'workspaceId': 'ws-1',
  };

  @override
  Future<List<WorkspaceMember>> listMembers(String id) async => [
    WorkspaceMember(
      userId: 'u-dana',
      username: 'dana',
      email: 'dana@test.local',
      role: 'member',
    ),
    WorkspaceMember(
      userId: 'u-sam',
      username: 'sam',
      email: 'sam@test.local',
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
  }) async {
    saved = {
      'title': title,
      'description': description,
      'status': status,
      'deadline': deadline,
      'acceptance': acceptance,
      'ownerId': ownerId,
    };
    return node;
  }

  static final node = RoadmapNode(
    id: 'n1',
    projectId: 'p1',
    title: 'Wire the flow',
    description: '',
    status: 'pending',
    position: 0,
  );
}

Future<void> _pumpCard(
  WidgetTester tester,
  ApiClient api,
  RoadmapNode node, {
  List<RoadmapNode>? all,
}) async {
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
            projectId: 'p1',
            node: node,
            onReload: () async {},
            childProgress: all == null
                ? null
                : RoadmapNodeCard.childDoneShare(all, node),
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets(
    'the edit dialog writes an acceptance sentence and picks the owner',
    (tester) async {
      final api = _MembersApi();
      await _pumpCard(tester, api, _MembersApi.node);

      await tester.tap(find.byType(PopupMenuButton<String>));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Edit').last);
      await tester.pumpAndSettle();

      await tester.enterText(
        find.byKey(const Key('acceptance-field')),
        'healthz answers 200',
      );
      await tester.pump();
      await tester.tap(find.byKey(const Key('owner-picker')));
      await tester.pumpAndSettle();
      await tester.tap(find.text('dana').last);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Save'));
      await tester.pumpAndSettle();

      expect(api.saved?['acceptance'], 'healthz answers 200');
      expect(api.saved?['ownerId'], 'u-dana');
    },
  );

  testWidgets('a leaf shows no progress bar; a parent shows its done share', (
    tester,
  ) async {
    final leaf = _MembersApi.node;
    final parent = RoadmapNode(
      id: 'r1',
      projectId: 'p1',
      parentId: null,
      title: 'Parent',
      description: '',
      status: 'in_progress',
      position: 0,
    );
    final childA = RoadmapNode(
      id: 'c1',
      projectId: 'p1',
      parentId: 'r1',
      title: 'A',
      description: '',
      status: 'done',
      position: 0,
    );
    final childB = RoadmapNode(
      id: 'c2',
      projectId: 'p1',
      parentId: 'r1',
      title: 'B',
      description: '',
      status: 'pending',
      position: 1,
    );

    await _pumpCard(tester, _MembersApi(), leaf);
    expect(find.byType(LinearProgressIndicator), findsNothing);

    await _pumpCard(
      tester,
      _MembersApi(),
      parent,
      all: [parent, childA, childB],
    );
    expect(find.byType(LinearProgressIndicator), findsOneWidget);
  });

  testWidgets('the progress view lists done items with owner and acceptance', (
    tester,
  ) async {
    final progress = RoadmapProgress(
      now: const [],
      needsDeadline: const [],
      next: const [],
      doneCount: 2,
      doneItems: [
        ProgressItem(
          id: 'd1',
          title: 'Sketch copy',
          status: 'done',
          ownerName: 'sam',
          acceptance: 'copy approved',
        ),
        ProgressItem(
          id: 'd2',
          title: 'Wire flow',
          status: 'done',
          ownerName: 'dana',
          acceptance: 'healthz 200',
        ),
      ],
    );
    await tester.pumpWidget(
      ChangeNotifierProvider(
        create: (_) =>
            Appearance(store: MemorySettingsStore(), locale: 'en-US'),
        child: MaterialApp(
          home: Scaffold(
            body: RoadmapProgressView(progress: progress, emptyLabel: 'empty'),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('DONE'), findsOneWidget);
    expect(find.text('Sketch copy'), findsOneWidget);
    expect(find.text('Owner: sam'), findsOneWidget);
    expect(find.textContaining('copy approved'), findsOneWidget);
    expect(find.text('Wire flow'), findsOneWidget);
    expect(find.text('Owner: dana'), findsOneWidget);
  });
}
