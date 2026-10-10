import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/roadmap_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// The two live streams (progress, presence) retry on their own clocks. When
// the network drops both fail together, so both sit in a backoff at once.
// Each stream plays a script: one HTTP status per call, then it stays open.
class _ScriptedApi extends ApiClient {
  _ScriptedApi({this.progressScript = const [], this.presenceScript = const []})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final List<int> progressScript;
  final List<int> presenceScript;
  int progressCalls = 0;
  int presenceCalls = 0;
  final _open = Completer<void>(); // an open stream, with no timers

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => [];

  @override
  Future<RoadmapProgress> roadmapProgress(String projectId) async =>
      RoadmapProgress(
        now: const [],
        needsDeadline: const [],
        next: const [],
        doneCount: 0,
      );

  @override
  Future<void> progressStream(
    String projectId,
    void Function(RoadmapProgress progress) onProgress, {
    Future<void>? stop,
  }) async {
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
    final call = presenceCalls++;
    if (call < presenceScript.length) {
      throw ApiException('down', status: presenceScript[call]);
    }
    await _open.future;
  }
}

Future<void> _pumpPage(WidgetTester tester, _ScriptedApi api) {
  final session = Session()
    ..ready = true
    ..api = api;
  return tester.pumpWidget(
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
}

void main() {
  testWidgets('leaving the page cancels the backoff timer of both streams', (
    tester,
  ) async {
    final api = _ScriptedApi(progressScript: [500], presenceScript: [500]);
    await _pumpPage(tester, api);
    await tester.pump(); // both streams failed: two backoff timers are live

    expect(api.progressCalls, 1);
    expect(api.presenceCalls, 1);

    // Disposing must leave no timer behind: flutter_test fails a test that
    // ends with one pending, which is what an orphaned second timer does.
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('a dropped stream reopens after 500ms, then waits 1s', (
    tester,
  ) async {
    final api = _ScriptedApi(progressScript: [500, 500]);
    await _pumpPage(tester, api);
    await tester.pump();
    expect(api.progressCalls, 1);

    await tester.pump(const Duration(milliseconds: 499));
    expect(api.progressCalls, 1);
    await tester.pump(const Duration(milliseconds: 1));
    await tester.pump();
    expect(api.progressCalls, 2);

    await tester.pump(const Duration(milliseconds: 999));
    expect(api.progressCalls, 2);
    await tester.pump(const Duration(milliseconds: 1));
    await tester.pump();
    expect(api.progressCalls, 3);

    // The third call stays open, so nothing else is scheduled.
    await tester.pumpWidget(const SizedBox());
  });

  for (final status in [401, 403, 404]) {
    testWidgets('a $status ends the retrying of both streams for good', (
      tester,
    ) async {
      final api = _ScriptedApi(
        progressScript: [status],
        presenceScript: [status],
      );
      await _pumpPage(tester, api);
      await tester.pump();
      await tester.pump(const Duration(seconds: 10));

      expect(api.progressCalls, 1);
      expect(api.presenceCalls, 1);
    });
  }
}
