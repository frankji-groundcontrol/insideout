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

class _FakeApi extends ApiClient {
  _FakeApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  int streams = 0;
  final _hold = Completer<void>();

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
    if (streams == 1) {
      throw ApiException('presence failed', status: 500);
    }
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
}
