import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/api/sse.dart';
import 'package:insideout/features/prd/coach_panel.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

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
    CoachHandlers handlers, {
    Future<void>? stop,
  }) async {
    applyCoachFrame(SseFrame('fact_recorded', payload), handlers);
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

void main() {
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
}
