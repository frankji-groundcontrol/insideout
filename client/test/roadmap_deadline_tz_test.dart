import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_node_card.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:provider/provider.dart';

// rv4-COMP-02: the deadline picker sent the picked day's local midnight as a
// UTC instant (`picked.toUtc()`), and every deadline label shows the first ten
// characters of the UTC string (`isoDate`). In any zone ahead of UTC the local
// midnight of 2026-10-18 is the evening of 2026-10-17 in UTC, so the dialog,
// the card and the progress list read the day before the one that was picked.
//
// This drives the real edit dialog and its real date picker: pick a day, read
// what the dialog shows, save, read what was sent, and read it back the way
// the node card does. Run it where the bug lives, as the wave gate does:
//   TZ=Asia/Shanghai flutter test test/roadmap_deadline_tz_test.dart
// It must also pass under TZ=UTC and TZ=America/New_York.

class _Api extends ApiClient {
  _Api()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  /// The `deadline` of every PATCH, as it goes on the wire.
  final sent = <String?>[];

  // No workspaceId: the dialog skips the member lookup.
  @override
  Future<Map<String, dynamic>> getProject(String id) async => {'id': id};

  @override
  Future<RoadmapNode> updateRoadmapNode(
    String nodeId, {
    String? title,
    String? description,
    String? status,
    String? deadline,
    String? acceptance,
    String? ownerId,
  }) async {
    sent.add(deadline);
    return _node(deadline);
  }
}

RoadmapNode _node(String? deadline) => RoadmapNode(
  id: 'n1',
  projectId: 'p1',
  title: 'Ship it',
  description: '',
  status: 'pending',
  position: 0,
  deadline: deadline,
);

Widget _host(Session session, RoadmapNode node) => MultiProvider(
  providers: [
    ChangeNotifierProvider.value(value: session),
    ChangeNotifierProvider(
      create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US'),
    ),
  ],
  child: MaterialApp(
    home: Scaffold(
      body: RoadmapNodeCard(projectId: 'p1', onReload: () async {}, node: node),
    ),
  ),
);

String _two(int n) => n.toString().padLeft(2, '0');

void main() {
  testWidgets('the picked deadline day is the day that is shown and sent', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(420, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final api = _Api();
    final session = Session()
      ..ready = true
      ..api = api;
    await tester.pumpWidget(_host(session, _node(null)));
    await tester.tap(find.byType(PopupMenuButton<String>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Edit').last);
    await tester.pumpAndSettle();

    // The picker opens on a week from now and OK picks that calendar day.
    final day = DateTime.now().add(const Duration(days: 7));
    final picked = '${day.year}-${_two(day.month)}-${_two(day.day)}';
    await tester.tap(find.widgetWithText(BlockButton, 'Set deadline'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('OK'));
    await tester.pumpAndSettle();

    final button = tester.widget<BlockButton>(
      find.byWidgetPredicate(
        (w) => w is BlockButton && w.icon == Icons.event_outlined,
      ),
    );
    expect(button.label, picked, reason: 'the dialog shows the picked day');

    await tester.tap(find.widgetWithText(BlockButton, 'Save'));
    await tester.pumpAndSettle();

    final wire = api.sent.single!;
    expect(
      wire,
      matches(RegExp(r'^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z$')),
      reason: 'the server parses RFC3339 only',
    );
    expect(
      wire.substring(0, 10),
      picked,
      reason: 'the day on the wire is the picked day',
    );

    // What the server would return for that instant, on the node card.
    await tester.pumpWidget(_host(session, _node(wire)));
    expect(find.text(picked), findsOneWidget);
  });
}
