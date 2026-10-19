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
import 'package:insideout/features/project/project_updates_feed.dart';
import 'package:insideout/features/project/roadmap_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_button.dart';
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

// rv4-FR-F02, F08, F15 ---------------------------------------------------------
// The updates feed had no in-flight guard on Accept/Reject: a double tap sent
// two decisions, the server adds a history note per decision, and Accept then
// Reject flipped a decision whose nodes were already built. The buttons stayed
// live until the reload replaced them. A fake api holds each decision on a
// completer, so the taps land while it is pending.

class _DecideApi extends ApiClient {
  _DecideApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final calls = <({String id, bool accept})>[];
  final answers = <Completer<Map<String, dynamic>>>[];
  int reloads = 0;

  @override
  Future<Map<String, dynamic>> decideProposal(
    String updateId, {
    required bool accept,
    String reason = '',
    bool apply = true,
  }) {
    calls.add((id: updateId, accept: accept));
    final answer = Completer<Map<String, dynamic>>();
    answers.add(answer);
    return answer.future;
  }
}

ProjectUpdate _proposal(String id, {String decision = ''}) => ProjectUpdate(
  id: id,
  kind: 'agent_proposal',
  content: 'Propose $id',
  createdAt: '2026-10-01T00:00:00Z',
  authorId: 'agent',
  decision: decision,
);

Future<void> _pumpFeed(
  WidgetTester tester,
  _DecideApi api,
  List<ProjectUpdate> updates,
) async {
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
          body: SingleChildScrollView(
            child: ProjectUpdatesFeed(
              projectId: 'p1',
              updates: updates,
              onChanged: () async => api.reloads++,
              onEdit: (u, content) async {},
              onRemove: (u) async {},
            ),
          ),
        ),
      ),
    ),
  );
}

BlockButton _button(WidgetTester tester, String label) =>
    tester.widget<BlockButton>(find.widgetWithText(BlockButton, label));

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

  group('rv4-FR-F02, F08, F15: a proposal is decided once', () {
    testWidgets('two taps on Accept before the answer send one decision', (
      tester,
    ) async {
      final api = _DecideApi();
      await _pumpFeed(tester, api, [_proposal('up1')]);

      // No frame between the taps: the second one reaches a live button.
      await tester.tap(find.text('Accept'));
      await tester.tap(find.text('Accept'));
      await tester.pump();

      expect(
        api.calls,
        hasLength(1),
        reason: 'each decision adds a history note every member reads',
      );
    });

    testWidgets('Reject while an Accept is pending is refused', (tester) async {
      final api = _DecideApi();
      await _pumpFeed(tester, api, [_proposal('up1')]);

      await tester.tap(find.text('Accept'));
      await tester.pump();
      expect(_button(tester, 'Accept').busy, isTrue);
      expect(_button(tester, 'Reject').onPressed, isNull);

      await tester.tap(find.text('Reject'));
      await tester.pump();

      expect(api.calls, [(id: 'up1', accept: true)]);
    });

    testWidgets('a decision that landed keeps the buttons off until the list '
        'shows it', (tester) async {
      final api = _DecideApi();
      await _pumpFeed(tester, api, [_proposal('up1')]);
      await tester.tap(find.text('Accept'));
      await tester.pump();

      api.answers.single.complete({'decision': 'accepted'});
      await tester.pump();
      await tester.pump();

      // The page reloads, but the list this feed holds is still the old one.
      expect(api.reloads, 1);
      expect(_button(tester, 'Accept').onPressed, isNull);
      expect(_button(tester, 'Reject').onPressed, isNull);
      await tester.tap(find.text('Reject'));
      await tester.pump();
      expect(api.calls, hasLength(1));
    });

    testWidgets('a refused decision puts the buttons back', (tester) async {
      final api = _DecideApi();
      await _pumpFeed(tester, api, [_proposal('up1')]);
      await tester.tap(find.text('Accept'));
      await tester.pump();

      api.answers.single.completeError(
        ApiException('unavailable', status: 503),
      );
      await tester.pump();
      await tester.pump();

      expect(api.reloads, 0);
      expect(_button(tester, 'Accept').busy, isFalse);
      expect(_button(tester, 'Accept').onPressed, isNotNull);
      expect(_button(tester, 'Reject').onPressed, isNotNull);
      await tester.tap(find.text('Reject'));
      await tester.pump();
      expect(api.calls, hasLength(2));
    });

    testWidgets('deciding one proposal leaves the others live', (tester) async {
      final api = _DecideApi();
      await _pumpFeed(tester, api, [_proposal('up1'), _proposal('up2')]);

      await tester.tap(find.text('Accept').first);
      await tester.pump();
      await tester.tap(find.text('Accept').last);
      await tester.pump();

      expect(api.calls.map((c) => c.id), ['up1', 'up2']);
    });

    testWidgets('a proposal that already carries a decision has no buttons', (
      tester,
    ) async {
      await _pumpFeed(tester, _DecideApi(), [
        _proposal('up1', decision: 'accepted'),
      ]);

      expect(find.text('Accept'), findsNothing);
      expect(find.text('Reject'), findsNothing);
    });
  });
}
