import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/project_page.dart';
import 'package:insideout/features/project/roadmap_canvas.dart';
import 'package:insideout/features/project/roadmap_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// Every animation under features/project honors the OS reduce-motion setting
// (MediaQuery.disableAnimations): the Evidence header flash and the minimap's
// scroll-to-band. Each test runs the same flow with the setting off as a
// control, so it fails if the motion is simply never there.

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
  Future<Map<String, dynamic>> getProject(String id) async => {
    'id': id,
    'workspaceId': 'ws1',
    'title': 'Motion',
    'description': '',
    'status': 'active',
    'repoUrl': '',
    'ownerId': 'u1',
    'updates': <Map<String, dynamic>>[],
  };

  /// Three root bands: 3 x 300 + gaps outgrow an 800px viewport.
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

ScrollPosition _canvasPosition(WidgetTester tester) => tester
    .state<ScrollableState>(
      find.descendant(
        of: find.byType(RoadmapCanvas),
        matching: find.byWidgetPredicate(
          (w) => w is Scrollable && w.axis == Axis.horizontal,
        ),
      ),
    )
    .position;

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

  for (final reduce in [false, true]) {
    testWidgets('minimap jump ${reduce ? 'is instant when' : 'glides unless'} '
        'reduce motion is on', (tester) async {
      await _pump(
        tester,
        const RoadmapPage(projectId: 'p1'),
        reduceMotion: reduce,
      );
      await tester.tap(find.text('Canvas'));
      await tester.pumpAndSettle();
      final position = _canvasPosition(tester);
      expect(position.maxScrollExtent, greaterThan(0));
      expect(position.pixels, 0);

      await tester.tap(
        find.descendant(
          of: find.byType(RoadmapMinimap),
          matching: find.text('R3'),
        ),
      );
      await tester.pump();

      if (reduce) {
        expect(position.pixels, position.maxScrollExtent);
      } else {
        expect(position.pixels, lessThan(position.maxScrollExtent));
        await tester.pumpAndSettle();
        expect(position.pixels, position.maxScrollExtent);
      }
    });
  }
}
