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

// The Bands tab and the Progress tab must tell the same story. The page
// hands Bands the live stream's reading, which a reload clears and a quiet or
// failed stream never refills; Bands must then use the one-shot reading the
// Progress tab falls back to, not its own status sort.
class _FakeApi extends ApiClient {
  _FakeApi(this.nodes, this.progress)
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final List<RoadmapNode> nodes;
  final RoadmapProgress progress;
  final _quiet = Completer<void>(); // the streams stay open and send nothing

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => nodes;

  @override
  Future<RoadmapProgress> roadmapProgress(String projectId) async => progress;

  @override
  Future<void> progressStream(
    String projectId,
    void Function(RoadmapProgress progress) onProgress, {
    Future<void>? stop,
  }) => _quiet.future;

  @override
  Future<void> presenceStream(
    String projectId,
    String clientId,
    void Function(List<PresenceEntry> entries) onChange, {
    void Function(String sessionId, String name, double x, double y)? onCursor,
    Future<void>? stop,
  }) => _quiet.future;
}

RoadmapNode _node(String id, String title, String status, int position) =>
    RoadmapNode(
      id: id,
      projectId: 'p1',
      title: title,
      description: '',
      status: status,
      position: position,
    );

Future<void> _openBands(
  WidgetTester tester,
  List<RoadmapNode> nodes,
  RoadmapProgress progress,
) async {
  final session = Session();
  session.ready = true;
  session.api = _FakeApi(nodes, progress);
  await tester.binding.setSurfaceSize(const Size(1400, 900));
  addTearDown(() => tester.binding.setSurfaceSize(null));

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
  await tester.tap(find.text('Bands'));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('Bands does not file finished work under Now', (tester) async {
    await _openBands(
      tester,
      [_node('d', 'Shipped thing', 'done', 0)],
      RoadmapProgress(
        now: const [],
        needsDeadline: const [],
        next: const [],
        doneCount: 1,
      ),
    );

    double dy(String text) => tester.getTopLeft(find.text(text)).dy;
    expect(dy('Shipped thing'), greaterThan(dy('LATER')));
  });

  testWidgets('Bands keeps an undated in-progress item out of Now, as the '
      'Progress tab does', (tester) async {
    final docs = _node('w', 'Docs pass', 'in_progress', 0);
    await _openBands(
      tester,
      [docs],
      RoadmapProgress(
        now: const [],
        needsDeadline: [
          ProgressItem(id: docs.id, title: docs.title, status: docs.status),
        ],
        next: const [],
        doneCount: 0,
      ),
    );

    double dy(String text) => tester.getTopLeft(find.text(text)).dy;
    expect(dy('Docs pass'), greaterThan(dy('LATER')));
  });
}
