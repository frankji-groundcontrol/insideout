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

// FEAT-05 behavioral coverage: the roadmap's default view renders the
// server's progress truth (deadlined Now with pressure, needsDeadline,
// justified Next, doneCount) instead of status-bucketed bands.
class _FakeApi extends ApiClient {
  _FakeApi() : super(
            baseUrl: 'http://127.0.0.1:9/api/v1',
            readAccess: () async => null,
            readRefresh: () async => null,
            writeTokens: (a, r) async {},
            clearTokens: () async {},
          );

  final _presenceParked = Completer<void>();

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => [];

  @override
  Future<RoadmapProgress> roadmapProgress(String projectId) async => RoadmapProgress.fromJson({
        'now': [
          {'id': 'n1', 'title': 'Payment retry', 'status': 'in_progress', 'deadline': '2026-10-06T00:00:00Z', 'pressure': 'near'},
          {'id': 'n2', 'title': 'Legacy migration', 'status': 'in_progress', 'deadline': '2026-09-01T00:00:00Z', 'pressure': 'overdue'},
        ],
        'needsDeadline': [
          {'id': 'n3', 'title': 'Docs pass', 'status': 'in_progress'},
        ],
        'next': [
          {'id': 'n4', 'title': 'Load test', 'status': 'pending', 'deadline': '2026-11-01T00:00:00Z', 'pressure': ''},
        ],
        'doneCount': 7,
      });

  @override
  Future<void> presenceStream(String projectId, String clientId, void Function(List<PresenceEntry> entries) onChange,
      {void Function(String sessionId, String name, double x, double y)? onCursor}) {
    // Park the presence loop without timers so the test disposes cleanly.
    return _presenceParked.future;
  }
}

void main() {
  testWidgets('roadmap defaults to the progress view with server-truth buckets', (tester) async {
    final session = Session();
    session.ready = true;
    session.api = _FakeApi();
    await tester.binding.setSurfaceSize(const Size(1400, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: session),
          ChangeNotifierProvider(create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US')),
          ChangeNotifierProvider(create: (_) => LoopFocus()),
        ],
        child: const MaterialApp(home: RoadmapPage(projectId: 'p1')),
      ),
    );
    await tester.pumpAndSettle();

    // The three server-truth bands render, not the status buckets.
    expect(find.text('NOW'), findsOneWidget);
    expect(find.text('NEEDS DEADLINE'), findsOneWidget);
    expect(find.text('NEXT'), findsOneWidget);
    expect(find.text('LATER'), findsNothing);

    // Items land in their server-assigned band with deadline and pressure.
    expect(find.text('Payment retry'), findsOneWidget);
    expect(find.text('2026-10-06'), findsOneWidget);
    expect(find.text('near'), findsOneWidget);
    expect(find.text('overdue'), findsOneWidget);
    expect(find.text('Docs pass'), findsOneWidget);

    // Done is counted, not listed.
    expect(find.text('7 done'), findsOneWidget);

    // Bands and canvas remain reachable as secondary modes.
    expect(find.text('Progress'), findsOneWidget);
    expect(find.text('Bands'), findsOneWidget);
    expect(find.text('Canvas'), findsOneWidget);
  });
}
