import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/api/sse.dart';
import 'package:insideout/features/prd/coach_panel.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// A coach turn holds its connection (and a model call) open for as long as it
// runs, and nothing closed it when the member left the PRD. sendCoach takes a
// `stop` for that; the panel owns it: every send gets the same future, and
// the panel completes it when it is disposed.
class _HoldingApi extends ApiClient {
  _HoldingApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  /// What the panel handed the send to stop it with.
  Future<void>? stop;
  final gate = Completer<void>();

  @override
  Future<void> sendCoach(
    String conversationId,
    String content,
    CoachHandlers handlers, {
    Future<void>? stop,
  }) async {
    this.stop = stop;
    await gate.future;
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

void main() {
  testWidgets('a send still streaming is stopped when the panel is gone', (
    tester,
  ) async {
    final api = _HoldingApi();
    await _pumpCoach(tester, api);

    await tester.enterText(find.byType(TextField), 'hello');
    await tester.tap(find.byIcon(Icons.send));
    await tester.pump();

    final stop = api.stop;
    expect(stop, isNotNull, reason: 'the panel gave the send no stop at all');
    var stopped = false;
    stop!.then((_) => stopped = true);

    // Folding the panel shut is not leaving it: the reply keeps coming.
    await tester.tap(find.byTooltip('Close the coach'));
    await tester.pump();
    expect(stopped, isFalse, reason: 'folding the panel cut the reply off');

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump();
    expect(stopped, isTrue, reason: 'the stream outlived its panel');

    api.gate.complete();
    await tester.pump();
    expect(tester.takeException(), isNull);
  });
}
