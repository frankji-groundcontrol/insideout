import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/project_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';
import '_project_model_helpers.dart';

// Every animation under features/project honors the OS reduce-motion setting
// (MediaQuery.disableAnimations): the Evidence header flash. Each test runs
// the same flow with the setting off as a control, so it fails if the motion
// is simply never there.

class _Api extends ApiClient {
  _Api()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final _parked = Completer<void>();

  @override
  Future<ProjectDetail> getProject(String id) async => projectDetail({
    'id': id,
    'workspaceId': 'ws1',
    'title': 'Motion',
    'description': '',
    'status': 'active',
    'repoUrl': '',
    'ownerId': 'u1',
    'updates': <Map<String, dynamic>>[],
  });

  /// The project page reads the roadmap for its leaf evidence.
  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => [
    for (var i = 0; i < 3; i++)
      RoadmapNode(
        id: 'R${i + 1}',
        projectId: projectId,
        title: 'R${i + 1}',
        description: '',
        status: 'pending',
        position: i,
      ),
  ];

  @override
  Future<RoadmapProgress> roadmapProgress(String projectId) async =>
      RoadmapProgress(now: [], needsDeadline: [], next: [], doneCount: 0);

  @override
  Future<void> progressStream(
    String projectId,
    void Function(RoadmapProgress progress) onProgress, {
    Future<void>? stop,
  }) => _parked.future;

  @override
  Future<void> presenceStream(
    String projectId,
    String clientId,
    void Function(List<PresenceEntry> entries) onChange, {
    void Function(String sessionId, String name, double x, double y)? onCursor,
    Future<void>? stop,
  }) => _parked.future;
}

Future<LoopFocus> _pump(
  WidgetTester tester,
  Widget page, {
  required bool reduceMotion,
}) async {
  final session = Session()
    ..ready = true
    ..api = _Api();
  final focus = LoopFocus();
  await tester.binding.setSurfaceSize(const Size(800, 900));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'en-US'),
        ),
        ChangeNotifierProvider.value(value: focus),
      ],
      child: MaterialApp(
        home: Builder(
          builder: (context) => MediaQuery(
            data: MediaQuery.of(
              context,
            ).copyWith(disableAnimations: reduceMotion),
            child: page,
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return focus;
}

/// Alpha of the Evidence header's flash border as currently painted.
double _flashAlpha(WidgetTester tester) {
  final box = tester.widget<DecoratedBox>(
    find
        .descendant(
          of: find.byKey(const ValueKey('evidence-pulse-flash')),
          matching: find.byType(DecoratedBox),
        )
        .first,
  );
  return ((box.decoration as BoxDecoration).border! as Border).top.color.a;
}

void main() {
  testWidgets('evidence flash animates by default', (tester) async {
    final focus = await _pump(
      tester,
      const ProjectPage(id: 'p1'),
      reduceMotion: false,
    );

    focus.pulseEvidence();
    await tester.pump();
    expect(_flashAlpha(tester), 0, reason: 'still fading in on frame one');

    await tester.pump(const Duration(milliseconds: 200));
    expect(_flashAlpha(tester), 1);
    await tester.pump(const Duration(seconds: 2));
  });

  testWidgets('reduce motion: the evidence flash lands without a fade', (
    tester,
  ) async {
    final focus = await _pump(
      tester,
      const ProjectPage(id: 'p1'),
      reduceMotion: true,
    );

    focus.pulseEvidence();
    await tester.pump();

    expect(_flashAlpha(tester), 1);
    await tester.pump(const Duration(seconds: 2));
  });
}
