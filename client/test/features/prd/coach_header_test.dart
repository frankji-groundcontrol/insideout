import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/prd/coach_panel.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// The masthead moved out of the panel into CoachHeader; nothing pinned its
// stage line or the close/open round trip, which now cross a widget boundary.
class _IdleApi extends ApiClient {
  _IdleApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );
}

Future<void> _pump(WidgetTester tester, {required bool conversation}) async {
  await tester.binding.setSurfaceSize(const Size(800, 1000));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  final session = Session()
    ..ready = true
    ..api = _IdleApi();
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
            conversation: conversation
                ? Conversation(
                    id: 'c1',
                    prdId: 'prd-1',
                    stage: 'critique',
                    status: 'active',
                  )
                : null,
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
  testWidgets(
    'the header names the stage and its close block folds the panel',
    (tester) async {
      await _pump(tester, conversation: true);

      expect(find.text('PRD COACH'), findsOneWidget);
      expect(find.text('Reviewing'), findsOneWidget);

      await tester.tap(find.byTooltip('Close the coach'));
      await tester.pump();
      expect(find.text('PRD COACH'), findsNothing);

      await tester.tap(find.byTooltip('Open the coach'));
      await tester.pump();
      expect(find.text('PRD COACH'), findsOneWidget);
    },
  );

  testWidgets('before a conversation exists the header shows no stage', (
    tester,
  ) async {
    await _pump(tester, conversation: false);

    expect(find.text('PRD COACH'), findsOneWidget);
    // The stage defaults to clarify, but with no conversation it has no meaning.
    expect(find.text('Clarifying'), findsNothing);
  });
}
