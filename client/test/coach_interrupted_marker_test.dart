import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/prd/coach_panel.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// A turn cut off by refresh or disconnect persists the stable English token
// "[interrupted]" on the server (agent/coach.go keeps it as the wire marker);
// the panel must not show that token verbatim to a zh reader — the marker is
// presentation, so it renders in the reader's locale (USER-08).
class _HistoryApi extends ApiClient {
  _HistoryApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );
}

CoachPanel _panel() => CoachPanel(
  prdId: 'prd-1',
  conversation: Conversation(
    id: 'c1',
    prdId: 'prd-1',
    stage: 'critique',
    status: 'active',
  ),
  messages: [
    ChatMessage(
      id: 'm1',
      role: 'assistant',
      content: 'partial answer\n\n[interrupted]',
    ),
  ],
  onPrdUpdated: () async {},
  onStageChanged: (_) {},
);

Future<void> _pump(WidgetTester tester, String locale) async {
  await tester.binding.setSurfaceSize(const Size(800, 1000));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  final session = Session()
    ..ready = true
    ..api = _HistoryApi();
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: locale),
        ),
      ],
      child: MaterialApp(home: Scaffold(body: _panel())),
    ),
  );
}

void main() {
  testWidgets('the interrupted marker renders localized, never the raw token', (
    tester,
  ) async {
    await _pump(tester, 'en-US');
    expect(find.textContaining('[interrupted]'), findsNothing);
    expect(find.textContaining('partial answer'), findsOneWidget);
  });

  testWidgets('a zh reader sees the zh marker', (tester) async {
    await _pump(tester, 'zh-CN');
    expect(find.textContaining('[interrupted]'), findsNothing);
    expect(find.textContaining('已中断'), findsOneWidget);
  });
}
