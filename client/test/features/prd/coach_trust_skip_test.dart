import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/api/sse.dart';
import 'package:insideout/features/prd/coach_panel.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// rv4-AIDEV-02: a critic pass that did not run opens the finalize gate. The
// person confirming the finalize has to be told, on the card and as it
// happens, that nothing was reviewed.
class _SkipApi extends ApiClient {
  _SkipApi({this.frame})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final SseFrame? frame;

  @override
  Future<void> sendCoach(
    String conversationId,
    String content,
    CoachHandlers handlers, {
    Future<void>? stop,
  }) async {
    if (frame != null) applyCoachFrame(frame!, handlers);
  }
}

Future<void> _pump(
  WidgetTester tester,
  Map<String, dynamic> conversation, {
  ApiClient? api,
  String locale = 'en-US',
}) async {
  final session = Session()
    ..ready = true
    ..api = api ?? _SkipApi();
  await tester.binding.setSurfaceSize(const Size(800, 1800));
  addTearDown(() => tester.binding.setSurfaceSize(null));
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
            conversation: Conversation.fromJson({
              'id': 'c1',
              'prdId': 'prd-1',
              'stage': 'critique',
              'status': 'active',
              ...conversation,
            }),
            messages: const [],
            onPrdUpdated: () async {},
            onStageChanged: (_) {},
            onConfirmStage: () async {},
            onDeclineStage: () {},
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('the finalize card says no critic review ran, and why', (
    tester,
  ) async {
    await _pump(tester, {
      'pendingStage': 'finalize',
      'criticSkipped': 'contention',
      'criticRounds': 0,
    });

    expect(find.textContaining('No critic review ran'), findsOneWidget);
    // The reason is words, not the wire token.
    expect(find.textContaining('the server was busy'), findsOneWidget);
    expect(find.textContaining('contention'), findsNothing);
  });

  testWidgets('a revision round that was skipped is named as such', (
    tester,
  ) async {
    await _pump(tester, {
      'pendingStage': 'finalize',
      'criticSkipped': 'critic-output-failure',
      'criticRounds': 1,
    });

    expect(find.textContaining('did not re-review'), findsOneWidget);
    expect(find.textContaining('No critic review ran'), findsNothing);
  });

  testWidgets('a reviewed PRD carries no skip note', (tester) async {
    await _pump(tester, {'pendingStage': 'finalize', 'criticRounds': 1});

    expect(find.byKey(const Key('coach-stage-confirm')), findsOneWidget);
    expect(find.textContaining('critic review'), findsNothing);
    expect(find.textContaining('re-review'), findsNothing);
  });

  testWidgets('a critic_skipped frame tells the person as it happens', (
    tester,
  ) async {
    await _pump(
      tester,
      const {},
      api: _SkipApi(
        frame: SseFrame('critic_skipped', '{"reason":"contention","rounds":0}'),
      ),
    );

    await tester.enterText(find.byType(TextField), 'finalize it');
    await tester.tap(find.byIcon(Icons.send));
    await tester.pumpAndSettle();

    expect(find.textContaining('No critic review ran'), findsOneWidget);
  });
}
