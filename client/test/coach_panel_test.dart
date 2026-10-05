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
  Future<void> sendCoach(String conversationId, String content, CoachHandlers handlers) async {
    const payload =
        '{"findings":[{"id":"c1","section":"goals","severity":"blocking","kind":"omission","issue":"no measurable objective","status":"open"},{"id":"c2","section":"users","severity":"minor","kind":"defect","issue":"segment is unnamed","quote":"someone","status":"open"}]}';
    applyCoachFrame(const SseFrame('critic_findings', payload), handlers);
  }
}

void main() {
  testWidgets('coach panel shows both critic findings from one SSE event', (tester) async {
    final session = Session();
    session.ready = true;
    session.api = _FindingsApi();

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: session),
          ChangeNotifierProvider(create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US')),
        ],
        child: MaterialApp(
          home: Scaffold(
            body: CoachPanel(
              prdId: 'prd-1',
              conversation: Conversation(id: 'c1', prdId: 'prd-1', stage: 'critique', status: 'active'),
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
    expect(find.textContaining('segment is unnamed'), findsOneWidget);
  });
}
