import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/api/sse.dart';
import 'package:insideout/features/prd/coach_column.dart';
import 'package:insideout/features/prd/coach_panel.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// The composer was cleared before the send finished, so a throttle, a 5xx or
// an idle timeout threw away what the member had written. A failed send puts
// the text back; a delivered one still empties the field; a suggestion chip
// never touches the field at all.
class _SendApi extends ApiClient {
  _SendApi(this.failure)
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  /// Thrown by every send; null means the send is delivered.
  final Object? failure;
  final sent = <String>[];

  @override
  Future<void> sendCoach(
    String conversationId,
    String content,
    CoachHandlers handlers, {
    Future<void>? stop,
  }) async {
    sent.add(content);
    final f = failure;
    if (f != null) throw f;
  }
}

Future<void> _pumpCoach(WidgetTester tester, ApiClient api) async {
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

String _field(WidgetTester tester) =>
    tester.widget<TextField>(find.byType(TextField)).controller!.text;

Future<void> _typeAndSend(WidgetTester tester, String text) async {
  await tester.enterText(find.byType(TextField), text);
  await tester.tap(find.byIcon(Icons.send));
  await tester.pump();
  await tester.pump();
}

void main() {
  final failures = <String, Object>{
    'a throttle': ApiException(
      'Rate limit exceeded',
      status: 429,
      code: 'APP_THROTTLE',
      retryAfterSeconds: 12,
    ),
    'a server error': ApiException('coach failed', status: 500),
    'an idle timeout': TimeoutException('no data'),
  };
  for (final entry in failures.entries) {
    testWidgets('${entry.key} puts the typed text back in the composer', (
      tester,
    ) async {
      final api = _SendApi(entry.value);
      await _pumpCoach(tester, api);

      await _typeAndSend(tester, 'hello coach');

      expect(api.sent, ['hello coach']);
      expect(_field(tester), 'hello coach');
    });
  }

  testWidgets('a failed send does not overwrite what was typed meanwhile', (
    tester,
  ) async {
    final gate = Completer<void>();
    final api = _GatedApi(gate);
    await _pumpCoach(tester, api);

    await _typeAndSend(tester, 'first draft');
    // The composer is locked while sending; a paste still lands in it.
    tester.widget<TextField>(find.byType(TextField)).controller!.text =
        'typed while waiting';
    gate.complete();
    await tester.pump();
    await tester.pump();

    expect(_field(tester), 'typed while waiting');
  });

  testWidgets('a delivered message still empties the composer', (tester) async {
    final api = _SendApi(null);
    await _pumpCoach(tester, api);

    await _typeAndSend(tester, 'hello coach');

    expect(api.sent, ['hello coach']);
    expect(_field(tester), isEmpty);
  });

  testWidgets('a suggestion chip leaves the typed text alone', (tester) async {
    final api = _SendApi(ApiException('coach failed', status: 500));
    await _pumpCoach(tester, api);

    await tester.enterText(find.byType(TextField), 'my own words');
    await tester.tap(find.byType(CoachPrompt).first);
    await tester.pump();
    await tester.pump();

    expect(api.sent, hasLength(1));
    expect(api.sent.single, isNot('my own words'));
    expect(_field(tester), 'my own words');
  });
}

// Fails once the test opens the gate.
class _GatedApi extends _SendApi {
  _GatedApi(this.gate) : super(null);

  final Completer<void> gate;

  @override
  Future<void> sendCoach(
    String conversationId,
    String content,
    CoachHandlers handlers, {
    Future<void>? stop,
  }) async {
    await gate.future;
    throw ApiException('coach failed', status: 500);
  }
}
