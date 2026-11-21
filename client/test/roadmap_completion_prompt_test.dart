import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_node_card.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

class _DoneApi extends ApiClient {
  _DoneApi(this.evidence)
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final List<Map<String, dynamic>> evidence;
  Map<String, dynamic>? saved;

  @override
  Future<Map<String, dynamic>> getProject(String id) async => {'id': id};

  @override
  Future<List<Map<String, dynamic>>> listNodeEvidence(String nodeId) async =>
      evidence;

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
    saved = {'status': status, 'completionReason': completionReason};
    return node;
  }

  static final node = RoadmapNode(
    canWrite: true,
    id: 'n1',
    projectId: 'p1',
    title: 'Ship the pilot',
    description: '',
    status: 'in_progress',
    position: 0,
  );
}

Future<void> _openStatus(WidgetTester tester, _DoneApi api) async {
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
            node: _DoneApi.node,
            onReload: () async {},
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
  await tester.tap(find.byKey(const Key('status-picker')));
  await tester.pumpAndSettle();
  await tester.tap(find.text('Done').last);
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('marking done with no evidence asks for one sentence', (
    tester,
  ) async {
    final api = _DoneApi(const []);
    await _openStatus(tester, api);

    expect(find.text('Why this is done'), findsOneWidget);
    await tester.enterText(find.byType(TextField).last, 'The pilot shipped.');
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(api.saved?['status'], 'done');
    expect(api.saved?['completionReason'], 'The pilot shipped.');
  });

  testWidgets('marking done with evidence does not ask for a reason', (
    tester,
  ) async {
    final api = _DoneApi(const [
      {'id': 'e1', 'kind': 'commit', 'detail': 'abc'},
    ]);
    await _openStatus(tester, api);

    expect(find.text('Why this is done'), findsNothing);
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(api.saved?['status'], 'done');
    expect(api.saved?['completionReason'], isNull);
  });
}
