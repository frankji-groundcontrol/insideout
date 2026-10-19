import 'dart:async';
import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/api/sse.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/roadmap_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

/// Whole-change review of the client API layer (rv4-FR-F01, F02, F08, F15).

// rv4-FR-F01 -----------------------------------------------------------------
// FULL-05 keeps the session after a refresh that merely failed (a dropped
// connection, a 5xx). The web streams then threw status 401 all the same, and
// the roadmap page's retry loop takes 401 for "signed out" and ends for good:
// live presence and progress stopped until a manual reload, with the user
// still signed in. The page is driven through its real loop here, over the
// fetch transport, with the wire scripted.

class _World {
  _World(this.boom);

  /// What the first refresh throws; later ones succeed.
  final Object Function() boom;
  int presenceOpens = 0;
  int refreshes = 0;
  bool cleared = false;

  static const _frame =
      'event: presence\ndata: [{"sessionId":"s1","userId":"u1","name":"mover"}]\n\n';

  /// The first presence open answers 401; every later one answers 200 with one
  /// presence frame and stays open. Progress is open and silent.
  Future<SseAttempt> fetch(
    String url,
    Map<String, String> headers, {
    Future<void>? stop,
  }) async {
    final open = StreamController<List<int>>();
    if (url.contains('/progress/')) {
      return SseAttempt(status: 200, chunks: open.stream);
    }
    presenceOpens++;
    if (presenceOpens == 1) {
      return const SseAttempt(status: 401, chunks: Stream.empty());
    }
    open.add(utf8.encode(_frame));
    return SseAttempt(status: 200, chunks: open.stream);
  }
}

class _StreamApi extends ApiClient {
  _StreamApi(_World world)
    : _world = world,
      super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => 'access-old',
        readRefresh: () async => 'refresh-1',
        writeTokens: (a, r) async {},
        clearTokens: () async => world.cleared = true,
      ) {
    fetchAttemptForTests = world.fetch;
  }

  final _World _world;

  @override
  Future<void> refresh() async {
    _world.refreshes++;
    if (_world.refreshes == 1) throw _world.boom();
  }

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => const [];

  @override
  Future<RoadmapProgress> roadmapProgress(String projectId) async =>
      RoadmapProgress(now: [], needsDeadline: [], next: [], doneCount: 0);
}

Future<void> _pumpRoadmap(WidgetTester tester, ApiClient api) async {
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
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: const MaterialApp(home: RoadmapPage(projectId: 'p1')),
    ),
  );
  await tester.pump();
}

void main() {
  group('rv4-FR-F01: a refresh that merely failed does not end the streams', () {
    for (final (name, boom) in <(String, Object Function())>[
      (
        'a dropped connection',
        () => DioException(
          requestOptions: RequestOptions(),
          type: DioExceptionType.connectionError,
        ),
      ),
      ('a 503', () => ApiException('unavailable', status: 503)),
    ]) {
      testWidgets('$name during refresh: the page reopens its presence stream', (
        tester,
      ) async {
        final world = _World(boom);
        await _pumpRoadmap(tester, _StreamApi(world));

        // The first open met a 401, the refresh failed, the session was kept.
        expect(world.presenceOpens, 1);
        expect(world.refreshes, 1);
        expect(world.cleared, isFalse);
        expect(find.text('mover'), findsNothing);

        // The retry loop backs off and opens again; a 401 would have ended it.
        await tester.pump(const Duration(milliseconds: 700));
        await tester.pump();

        expect(
          world.presenceOpens,
          2,
          reason: 'the loop took a kept session for a signed-out one',
        );
        expect(find.text('mover'), findsOneWidget);
        expect(world.cleared, isFalse);
      });
    }
  });
}
