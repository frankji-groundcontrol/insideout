import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/api/sse.dart';
import 'package:insideout/features/prd/coach_panel.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

/// Emits message_start as agent/sse.go messageStart writes it for the
/// template streamer: {id, model: "template", offline: true}.
class _OfflineApi extends ApiClient {
  _OfflineApi()
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
    CoachHandlers handlers, {
    Future<void>? stop,
  }) async {
    applyCoachFrame(
      SseFrame(
        'message_start',
        '{"id":"m1","model":"template","offline":true}',
      ),
      handlers,
    );
    applyCoachFrame(
      SseFrame('delta', '{"text":"we are two people"}'),
      handlers,
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

void main() {
  // USER-03: when the stream's message_start reports offline, the panel
  // badges it — the canned echo must never pass for real coaching. The
  // wording is localized like every panel string.
  for (final (locale, badge) in const [
    (
      'en-US',
      'AI coaching is offline on this server. Replies are echoes, not AI answers.',
    ),
    ('zh-CN', '此服务器未配置 AI，教练回复为原样回显，并非 AI 回答。'),
  ]) {
    testWidgets('an offline stream badges the coach panel ($locale)', (
      tester,
    ) async {
      await _pumpCoach(tester, _OfflineApi(), locale: locale);

      await tester.enterText(find.byType(TextField), 'who is it for?');
      await tester.tap(find.byIcon(Icons.send));
      await tester.pump();
      await tester.pumpAndSettle();

      expect(find.text(badge), findsOneWidget);
      // The echo itself still streams as the turn's reply.
      expect(find.text('we are two people'), findsOneWidget);
    });
  }
}
