import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_node_card.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:provider/provider.dart';

// Clearing a deadline in the edit dialog must put the "Set deadline" label
// back, not leave an empty button.

class _Api extends ApiClient {
  _Api()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  // No workspaceId: the dialog skips the member lookup.
  @override
  Future<Map<String, dynamic>> getProject(String id) async => {'id': id};
}

void main() {
  testWidgets('clearing the deadline restores the Set deadline label', (
    tester,
  ) async {
    final session = Session()
      ..ready = true
      ..api = _Api();
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
            body: RoadmapNodeCard(
              projectId: 'p1',
              onReload: () async {},
              node: RoadmapNode(
                id: 'n1',
                projectId: 'p1',
                title: 'Ship it',
                description: '',
                status: 'pending',
                position: 0,
                deadline: '2026-12-01T00:00:00Z',
              ),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.byType(PopupMenuButton<String>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Edit').last);
    await tester.pumpAndSettle();
    expect(find.widgetWithText(BlockButton, '2026-12-01'), findsOneWidget);

    await tester.tap(find.byTooltip('Clear deadline'));
    await tester.pumpAndSettle();

    expect(find.widgetWithText(BlockButton, 'Set deadline'), findsOneWidget);
    expect(
      tester
          .widgetList<BlockButton>(find.byType(BlockButton))
          .where((b) => b.label.isEmpty),
      isEmpty,
    );
  });
}
