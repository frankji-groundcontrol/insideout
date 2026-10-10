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

// Two findings as agent/critic.go CriticFinding marshals them; the first
// carries a suggestion, the second does not.
const _findings =
    '{"findings":[{"id":"c1","section":"goals","severity":"blocking","kind":"omission","issue":"no measurable objective","suggestion":"name one metric and a target","status":"open"},{"id":"c2","section":"users","severity":"minor","kind":"defect","issue":"segment is unnamed","quote":"someone","status":"open"}]}';

class _FindingsApi extends ApiClient {
  _FindingsApi({this.payload = _findings, this.later})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final String payload;

  /// Applied instead of the findings on every send after the first.
  final SseFrame? later;
  int sends = 0;

  @override
  Future<void> sendCoach(
    String conversationId,
    String content,
    CoachHandlers handlers,
  ) async {
    sends += 1;
    final next = sends > 1 ? later : null;
    applyCoachFrame(next ?? SseFrame('critic_findings', payload), handlers);
  }
}

/// Emits fact_recorded as agent/sse.go factRecorded writes it:
/// {id, kind, text, status}, never a bare key/value pair.
class _FactApi extends ApiClient {
  _FactApi(this.payload)
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final String payload;

  @override
  Future<void> sendCoach(
    String conversationId,
    String content,
    CoachHandlers handlers,
  ) async {
    applyCoachFrame(SseFrame('fact_recorded', payload), handlers);
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

Future<void> _pumpCoach(
  WidgetTester tester,
  ApiClient api, {
  String locale = 'en-US',
}) async {
  final session = Session();
  session.ready = true;
  session.api = api;

  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: locale),
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

// Tall enough that every finding is on screen without scrolling.
Future<void> _pumpTall(
  WidgetTester tester,
  ApiClient api,
  String locale,
) async {
  await tester.binding.setSurfaceSize(const Size(800, 1800));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await _pumpCoach(tester, api, locale: locale);
}

Future<void> _send(WidgetTester tester, String text) async {
  await tester.enterText(find.byType(TextField), text);
  await tester.tap(find.byIcon(Icons.send));
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

  // The finding block rendered only the issue, under the raw English token
  // 'critic' (also in zh). Severity, kind and suggestion were sent and dropped.
  for (final (locale, blocking, minor, label, suggestion) in const [
    (
      'en-US',
      'Blocking',
      'Minor',
      'Review finding',
      'Suggestion: name one metric and a target',
    ),
    ('zh-CN', '阻塞', '次要', '评审发现', '建议：name one metric and a target'),
  ]) {
    testWidgets('a critic finding shows its severity, label and suggestion '
        '($locale)', (tester) async {
      await _pumpTall(tester, _FindingsApi(), locale);
      await _send(tester, 'review this');

      expect(find.text('$blocking / $label'), findsOneWidget);
      expect(find.text('$minor / $label'), findsOneWidget);
      // Only the first finding carries a suggestion.
      expect(find.text(suggestion), findsOneWidget);
      expect(find.text('critic'), findsNothing);
    });
  }

  testWidgets('an unknown severity prints the label alone, never the token', (
    tester,
  ) async {
    await _pumpTall(
      tester,
      _FindingsApi(
        payload:
            '{"findings":[{"id":"c1","section":"goals","severity":"catastrophic","issue":"no measurable objective","status":"open"}]}',
      ),
      'en-US',
    );
    await _send(tester, 'review this');

    expect(find.textContaining('no measurable objective'), findsOneWidget);
    expect(find.text('Review finding'), findsOneWidget);
    expect(find.textContaining('catastrophic'), findsNothing);
    expect(find.text('critic'), findsNothing);
  });

  // Findings belong to the stage they were raised in; the server keeps the list
  // across non-critique turns on purpose, so only a stage change retires them.
  testWidgets('a stage change retires the findings raised in the old stage', (
    tester,
  ) async {
    final api = _FindingsApi(
      later: const SseFrame('stage_changed', '{"stage":"finalize"}'),
    );
    await _pumpTall(tester, api, 'en-US');
    await _send(tester, 'review this');
    expect(find.textContaining('no measurable objective'), findsOneWidget);

    await _send(tester, 'looks good, finalize');

    expect(api.sends, 2);
    expect(find.textContaining('no measurable objective'), findsNothing);
    expect(find.textContaining('segment is unnamed'), findsNothing);
  });

  testWidgets('a send that leaves the stage alone keeps the findings', (
    tester,
  ) async {
    final api = _FindingsApi(
      later: const SseFrame('delta', '{"text":"Take the first one."}'),
    );
    await _pumpTall(tester, api, 'en-US');
    await _send(tester, 'review this');
    await _send(tester, 'which first?');

    expect(find.textContaining('Take the first one.'), findsOneWidget);
    expect(find.textContaining('no measurable objective'), findsOneWidget);
    expect(find.textContaining('segment is unnamed'), findsOneWidget);
  });

  // The panel interpolated the whole payload Map, so the snackbar read
  // "Fact: {id: f1, kind: constraint, text: ..., status: proposed}".
  for (final (locale, shown) in const [
    ('en-US', 'Fact: Solo founders only'),
    ('zh-CN', '事实：Solo founders only'),
  ]) {
    testWidgets('a recorded fact shows its text, not the raw payload ($locale)', (
      tester,
    ) async {
      await _pumpCoach(
        tester,
        _FactApi(
          '{"id":"f1","kind":"constraint","text":"Solo founders only","status":"proposed"}',
        ),
        locale: locale,
      );

      await tester.enterText(find.byType(TextField), 'we are two people');
      await tester.tap(find.byIcon(Icons.send));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 750));

      expect(find.text(shown), findsOneWidget);
      expect(find.textContaining('kind:'), findsNothing);
      // Let the snackbar's own timer run out inside the test clock.
      await tester.pump(const Duration(seconds: 5));
      await tester.pumpAndSettle();
    });
  }

  testWidgets('a recorded fact with no text shows no snackbar', (tester) async {
    await _pumpCoach(
      tester,
      _FactApi('{"id":"f1","kind":"constraint","status":"proposed"}'),
    );

    await tester.enterText(find.byType(TextField), 'we are two people');
    await tester.tap(find.byIcon(Icons.send));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 750));

    expect(find.byType(SnackBar), findsNothing);
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
