import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/roadmap_canvas.dart';
import 'package:insideout/features/project/roadmap_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

class _FakeApi extends ApiClient {
  _FakeApi({this.failFirstStream = true})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  /// The reconnect test wants the first stream to die; the pointer test does
  /// not, so it need not wait out the backoff.
  final bool failFirstStream;
  int streams = 0;
  final _hold = Completer<void>();

  // What the page handed the stream, kept so a test can play the server.
  String? clientId;
  void Function(List<PresenceEntry> entries)? onChange;
  void Function(String sessionId, String name, double x, double y)? onCursor;

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => [
    RoadmapNode.fromJson({
      'id': 'n1',
      'projectId': projectId,
      'title': 'Ship it',
      'description': '',
      'status': 'in_progress',
      'position': 0,
    }),
  ];

  @override
  Future<void> progressStream(
    String projectId,
    void Function(RoadmapProgress progress) onProgress, {
    Future<void>? stop,
  }) {
    return _hold.future;
  }

  @override
  Future<void> presenceStream(
    String projectId,
    String clientId,
    void Function(List<PresenceEntry> entries) onChange, {
    void Function(String sessionId, String name, double x, double y)? onCursor,
    Future<void>? stop,
  }) async {
    streams++;
    if (failFirstStream && streams == 1) {
      throw ApiException('presence failed', status: 500);
    }
    this.clientId = clientId;
    this.onChange = onChange;
    this.onCursor = onCursor;
    onChange([
      PresenceEntry.fromJson({
        'sessionId': 'other',
        'userId': 'u2',
        'name': 'mover',
      }),
    ]);
    await _hold.future; // stay "connected" without pending timers
  }
}

void main() {
  testWidgets('roadmap presence reconnects after a stream drop', (
    tester,
  ) async {
    final session = Session();
    session.ready = true;
    final fake = _FakeApi();
    session.api = fake;

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
    await tester.pump();

    // First stream died before delivering anything: no chips yet.
    expect(fake.streams, 1);
    expect(find.text('mover'), findsNothing);

    // Capped backoff elapses; the reopened stream resyncs the snapshot.
    await tester.pump(const Duration(milliseconds: 700));
    await tester.pump();

    expect(fake.streams, 2);
    expect(find.text('mover'), findsOneWidget);
  });

  // Multiplayer pointers: the page draws the other members' cursors on the
  // canvas from the stream's cursor events and keeps them in step with who is
  // present. Nothing else exercises that path.
  testWidgets(
    'another member\'s pointer follows them and goes when they leave',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));
      final session = Session();
      session.ready = true;
      final fake = _FakeApi(failFirstStream: false);
      session.api = fake;

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
      await tester.pump();
      await tester.tap(find.text('Canvas'));
      await tester.pumpAndSettle();

      // The presence chip also reads 'mover'; the pointer is the canvas's own.
      final pointer = find.descendant(
        of: find.byType(RoadmapCanvas),
        matching: find.byIcon(Icons.navigation),
      );
      expect(pointer, findsNothing);

      fake.onCursor!('other', 'mover', 120, 80);
      await tester.pump();
      expect(pointer, findsOneWidget);
      expect(
        find.descendant(
          of: find.byType(RoadmapCanvas),
          matching: find.text('mover'),
        ),
        findsOneWidget,
      );
      final first = tester.getTopLeft(pointer);

      // The same member moving moves their pointer; it does not add another.
      fake.onCursor!('other', 'mover', 220, 130);
      await tester.pump();
      expect(pointer, findsOneWidget);
      expect(tester.getTopLeft(pointer) - first, const Offset(100, 50));

      // The page's own cursor events come back over the stream: no pointer.
      fake.onCursor!(fake.clientId!, 'me', 5, 5);
      await tester.pump();
      expect(pointer, findsOneWidget);

      // The member leaves: their pointer goes with their presence chip.
      fake.onChange!([]);
      await tester.pump();
      expect(pointer, findsNothing);
    },
  );
}
