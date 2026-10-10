import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/roadmap_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

/// The Progress view only lists deadlined work (Now, Next), in-progress work
/// missing a deadline, and a done count. A roadmap whose goals are all
/// to-do and undated therefore comes back from the server with nothing to
/// show, and the page greeted the user's first goal with "No roadmap yet".
class _FakeApi extends ApiClient {
  _FakeApi({required this.nodes})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final List<RoadmapNode> nodes;
  final _parked = Completer<void>();

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => nodes;

  @override
  Future<RoadmapProgress> roadmapProgress(String projectId) async =>
      RoadmapProgress.fromJson({'doneCount': 0});

  @override
  Future<void> presenceStream(
    String projectId,
    String clientId,
    void Function(List<PresenceEntry> entries) onChange, {
    void Function(String sessionId, String name, double x, double y)? onCursor,
  }) => _parked.future;

  @override
  Future<void> progressStream(
    String projectId,
    void Function(RoadmapProgress progress) onProgress,
  ) => _parked.future;
}

Future<void> _pump(WidgetTester tester, List<RoadmapNode> nodes) async {
  final session = Session()
    ..ready = true
    ..api = _FakeApi(nodes: nodes);
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'en-US'),
        ),
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: const MaterialApp(home: RoadmapPage(projectId: 'p1')),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('goals that exist but are not scheduled are not "no roadmap"', (
    tester,
  ) async {
    await _pump(tester, [
      RoadmapNode.fromJson({
        'id': 'n1',
        'projectId': 'p1',
        'title': 'Ship the MVP',
        'status': 'pending',
      }),
    ]);

    expect(find.textContaining('No roadmap yet'), findsNothing);
    expect(find.textContaining('Nothing is scheduled yet'), findsOneWidget);
  });

  testWidgets('a project with no goals still says there is no roadmap', (
    tester,
  ) async {
    await _pump(tester, const []);

    expect(find.textContaining('No roadmap yet'), findsOneWidget);
    expect(find.textContaining('Nothing is scheduled yet'), findsNothing);
  });
}
