import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/api/sse.dart';
import 'package:insideout/features/prd/coach_panel.dart';
import 'package:insideout/l10n/l10n.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// The coach panel stored `e.message` (the raw server string) for an
// ApiException, so localizedError saw a plain String and collapsed every coach
// API failure to error.generic; the countdown copy then language-sniffed that
// English string for "unavailable". The panel keeps the exception object now,
// and both notices read its structured status and code. Pinned in both
// locales against the codes the server really sends (agent/coach.go).
class _FailingApi extends ApiClient {
  _FailingApi(this.failure)
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final ApiException failure;

  @override
  Future<void> sendCoach(
    String conversationId,
    String content,
    CoachHandlers handlers,
  ) async => throw failure;
}

// Fails only once the test says so, so the failure can land after the panel
// has left the screen.
class _LateFailingApi extends ApiClient {
  _LateFailingApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final gate = Completer<void>();

  @override
  Future<void> sendCoach(
    String conversationId,
    String content,
    CoachHandlers handlers,
  ) async {
    await gate.future;
    throw ApiException(
      'Rate limit exceeded',
      status: 429,
      code: 'APP_THROTTLE',
      retryAfterSeconds: 12,
    );
  }
}

// Keeps the handlers the panel registered and holds the send open, so a stream
// event can be delivered after the panel has left the screen.
class _HoldingApi extends ApiClient {
  _HoldingApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final gate = Completer<void>();
  CoachHandlers? handlers;

  @override
  Future<void> sendCoach(
    String conversationId,
    String content,
    CoachHandlers handlers,
  ) async {
    this.handlers = handlers;
    await gate.future;
  }
}

typedef _Case = ({
  String name,
  ApiException failure,
  String notice,
  String? countdown,
  int seconds,
});

final _cases = <_Case>[
  (
    name: 'app throttle',
    failure: ApiException(
      'Rate limit exceeded',
      status: 429,
      code: 'APP_THROTTLE',
      retryAfterSeconds: 12,
    ),
    notice: 'error.throttled',
    countdown: 'coach.rateLimited',
    seconds: 12,
  ),
  (
    // Its message happens to contain "unavailable".
    name: 'circuit open',
    failure: ApiException(
      'AI service temporarily unavailable',
      status: 503,
      code: 'CIRCUIT_OPEN',
    ),
    notice: 'error.throttled',
    countdown: 'coach.serviceUnavailable',
    seconds: 30,
  ),
  (
    // Same code, a message that does not: only the code can tell them apart.
    name: 'circuit busy',
    failure: ApiException(
      'AI service is busy, try again shortly',
      status: 503,
      code: 'CIRCUIT_OPEN',
    ),
    notice: 'error.throttled',
    countdown: 'coach.serviceUnavailable',
    seconds: 30,
  ),
  (
    // Arrives as an SSE error event: a code, no HTTP status.
    name: 'provider rate limit',
    failure: ApiException(
      'AI provider is rate limited',
      code: 'ANTHROPIC_RATE_LIMIT',
    ),
    notice: 'error.throttled',
    countdown: 'coach.rateLimited',
    seconds: 30,
  ),
  (
    name: 'server error',
    failure: ApiException('coach failed', status: 500),
    notice: 'error.server',
    countdown: null,
    seconds: 0,
  ),
  (
    name: 'forbidden',
    failure: ApiException('not your PRD', status: 403),
    notice: 'error.forbidden',
    countdown: null,
    seconds: 0,
  ),
];

Future<void> _pumpCoach(
  WidgetTester tester,
  ApiClient api,
  String locale,
) async {
  // Tall enough for the error notice and the countdown notice together.
  await tester.binding.setSurfaceSize(const Size(800, 1000));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  final session = Session()
    ..ready = true
    ..api = api;
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

void main() {
  // The ApiException branch set state and started the countdown timer without
  // asking whether the panel was still mounted: leaving the PRD while a send
  // was in flight, then a backoff failure, threw on a disposed State and left a
  // periodic timer running.
  testWidgets(
    'a backoff failure that lands after the panel is gone is dropped',
    (tester) async {
      final api = _LateFailingApi();
      await _pumpCoach(tester, api, 'en-US');

      await tester.enterText(find.byType(TextField), 'hello');
      await tester.tap(find.byIcon(Icons.send));
      await tester.pump();
      await tester.pumpWidget(const SizedBox.shrink());

      api.gate.complete();
      await tester.pump();
      await tester.pump(const Duration(seconds: 2));

      expect(tester.takeException(), isNull);
    },
  );

  // The delta callback was the one stream callback without a mounted guard: a
  // chunk landing after the member left the PRD set state on a disposed State.
  // Every callback must be a quiet no-op once the panel is gone.
  testWidgets('a stream event that lands after the panel is gone is dropped', (
    tester,
  ) async {
    final api = _HoldingApi();
    await _pumpCoach(tester, api, 'en-US');

    await tester.enterText(find.byType(TextField), 'hello');
    await tester.tap(find.byIcon(Icons.send));
    await tester.pump();
    await tester.pumpWidget(const SizedBox.shrink());

    final h = api.handlers!;
    expect(() => h.onDelta!('late'), returnsNormally);
    expect(() => h.onCriticFindings!(const []), returnsNormally);
    expect(() => h.onFactRecorded!({'text': 'late'}), returnsNormally);
    expect(() => h.onStageChanged!('finalize'), returnsNormally);

    api.gate.complete();
    await tester.pump();
    expect(tester.takeException(), isNull);
  });

  for (final locale in const ['en-US', 'zh-CN']) {
    String t(String key, [Map<String, Object?>? args]) =>
        lookup(locale, key, args);

    for (final c in _cases) {
      testWidgets('a ${c.name} failure shows its localized copy, not the '
          'generic one ($locale)', (tester) async {
        await _pumpCoach(tester, _FailingApi(c.failure), locale);

        await tester.enterText(find.byType(TextField), 'hello');
        await tester.tap(find.byIcon(Icons.send));
        await tester.pump();
        await tester.pump();

        // The notice is the status/code mapping, localized.
        expect(find.text(t(c.notice)), findsOneWidget);
        expect(find.text(t('error.generic')), findsNothing);
        // The raw server string never reaches the page.
        expect(find.text(c.failure.message), findsNothing);

        // Backoff codes also run a localized countdown, chosen by the code
        // and not by reading the English message; the composer stays locked
        // while it runs. Other failures leave the composer free.
        final countdown = c.countdown;
        if (countdown != null) {
          expect(
            find.text(t(countdown, {'seconds': c.seconds})),
            findsOneWidget,
          );
        }
        expect(
          tester.widget<TextField>(find.byType(TextField)).enabled,
          countdown == null,
        );
      });
    }
  }
}
