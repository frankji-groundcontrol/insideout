import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/api/sse.dart';
import 'package:insideout/features/prd/coach_panel.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

class _FindingsApi extends ApiClient {
  _FindingsApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  @override
  Future<void> sendCoach(
    String conversationId,
    String content,
    CoachHandlers handlers,
  ) async {
    const payload =
        '{"findings":[{"id":"c1","section":"goals","severity":"blocking","kind":"omission","issue":"no measurable objective","status":"open"},{"id":"c2","section":"users","severity":"minor","kind":"defect","issue":"segment is unnamed","quote":"someone","status":"open"}]}';
    applyCoachFrame(const SseFrame('critic_findings', payload), handlers);
  }
}

/// First send stalls like a server holding the SSE connection open: it
/// emits one delta after [firstEventDelay] (if any) and then never
/// completes. Later sends answer immediately so retries can be observed.
class _StallingApi extends ApiClient {
  _StallingApi({this.firstEventDelay = Duration.zero})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final Duration firstEventDelay;
  int sends = 0;

  @override
  Future<void> sendCoach(
    String conversationId,
    String content,
    CoachHandlers handlers,
  ) async {
    sends += 1;
    if (sends > 1) {
      handlers.onDelta?.call('recovered');
      return;
    }
    // No frames until [firstEventDelay], then silence, so the real 75s
    // readCoachSse idle timer — reset by any complete frame — is what ends
    // the send. A direct onDelta would skip that timer.
    final chunks = StreamController<List<int>>();
    if (firstEventDelay > Duration.zero) {
      Timer(firstEventDelay, () {
        if (chunks.isClosed) return;
        chunks.add(
          utf8.encode('event: delta\ndata: {"text":"partial answer"}\n\n'),
        );
      });
    }
    await readCoachSse(
      chunks.stream,
      handlers,
      idleTimeout: coachSseIdleTimeout,
    );
  }
}

Future<void> _pumpCoach(WidgetTester tester, ApiClient api) async {
  final session = Session();
  session.ready = true;
  session.api = api;

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
          body: CoachPanel(
            prdId: 'prd-1',
            conversation: Conversation(
              id: 'c1',
              prdId: 'prd-1',
              stage: 'critique',
              status: 'active',
            ),
            messages: const [],
            onPrdUpdated: () async {},
            onStageChanged: (_) {},
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('coach panel shows both critic findings from one SSE event', (
    tester,
  ) async {
    final session = Session();
    session.ready = true;
    session.api = _FindingsApi();

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
            body: CoachPanel(
              prdId: 'prd-1',
              conversation: Conversation(
                id: 'c1',
                prdId: 'prd-1',
                stage: 'critique',
                status: 'active',
              ),
              messages: const [],
              onPrdUpdated: () async {},
              onStageChanged: (_) {},
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField), 'review this');
    await tester.tap(find.byIcon(Icons.send));
    await tester.pumpAndSettle();

    expect(find.textContaining('no measurable objective'), findsOneWidget);
    await tester.drag(find.byType(ListView), const Offset(0, -400));
    await tester.pumpAndSettle();
    expect(find.textContaining('segment is unnamed'), findsOneWidget);
  });

  testWidgets(
    '75s idle stream surfaces the error and re-enables the composer',
    (tester) async {
      await _pumpCoach(tester, _StallingApi());

      await tester.enterText(find.byType(TextField), 'hello');
      await tester.tap(find.byIcon(Icons.send));
      await tester.pump();

      expect(find.textContaining('Network problem'), findsNothing);
      expect(tester.widget<TextField>(find.byType(TextField)).enabled, isFalse);

      await tester.pump(const Duration(seconds: 75));

      expect(find.textContaining('Network problem'), findsOneWidget);
      expect(tester.widget<TextField>(find.byType(TextField)).enabled, isTrue);
    },
  );

  testWidgets('a stream event resets the 75s idle timer', (tester) async {
    await _pumpCoach(
      tester,
      _StallingApi(firstEventDelay: const Duration(seconds: 74)),
    );

    await tester.enterText(find.byType(TextField), 'hello');
    await tester.tap(find.byIcon(Icons.send));
    await tester.pump();
    expect(find.textContaining('Network problem'), findsNothing);

    await tester.pump(const Duration(seconds: 74));
    expect(find.textContaining('partial answer'), findsOneWidget);
    expect(find.textContaining('Network problem'), findsNothing);

    // t=76s from send: past the original deadline, timer was reset by the event.
    await tester.pump(const Duration(seconds: 2));
    expect(find.textContaining('Network problem'), findsNothing);

    await tester.pump(const Duration(seconds: 75));
    expect(find.textContaining('Network problem'), findsOneWidget);
  });

  testWidgets('user can send again after the idle timeout', (tester) async {
    final api = _StallingApi();
    await _pumpCoach(tester, api);

    await tester.enterText(find.byType(TextField), 'hello');
    await tester.tap(find.byIcon(Icons.send));
    await tester.pump();
    await tester.pump(const Duration(seconds: 75));
    expect(find.textContaining('Network problem'), findsOneWidget);

    await tester.enterText(find.byType(TextField), 'again');
    await tester.tap(find.byIcon(Icons.send));
    await tester.pumpAndSettle();

    expect(find.textContaining('recovered'), findsOneWidget);
    expect(find.textContaining('Network problem'), findsNothing);
  });
}
