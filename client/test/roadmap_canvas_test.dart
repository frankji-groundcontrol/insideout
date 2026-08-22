import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_canvas.dart';
import 'package:insideout/features/project/roadmap_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

class _FakeApi extends ApiClient {
  _FakeApi() : super(
            baseUrl: 'http://127.0.0.1:9/api/v1',
            readAccess: () async => null,
            readRefresh: () async => null,
            writeTokens: (a, r) async {},
            clearTokens: () async {},
          );

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => [
        RoadmapNode(id: 'r1', projectId: 'p1', parentId: null, title: 'Band A', description: '', status: 'pending', position: 0),
        RoadmapNode(id: 'r2', projectId: 'p1', parentId: null, title: 'Band B', description: '', status: 'pending', position: 1),
        RoadmapNode(id: 'c1', projectId: 'p1', parentId: 'r1', title: 'A leaf', description: '', status: 'in_progress', position: 0),
      ];
}

void main() {
  testWidgets('roadmap page toggles to the canvas with bands and minimap', (tester) async {
    final session = Session();
    session.ready = true;
    session.api = _FakeApi();
    await tester.binding.setSurfaceSize(const Size(1400, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: session),
          ChangeNotifierProvider(create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US')),
        ],
        child: const MaterialApp(home: RoadmapPage(projectId: 'p1')),
      ),
    );
    await tester.pumpAndSettle();

    // List view first: both roots present, no canvas yet.
    expect(find.text('Band A'), findsWidgets);
    expect(find.byType(RoadmapMinimap), findsNothing);

    // Toggle to canvas.
    await tester.tap(find.text('Canvas'));
    await tester.pumpAndSettle();

    expect(find.byType(RoadmapMinimap), findsOneWidget);
    expect(find.byType(RoadmapCanvas), findsOneWidget);
    // Both sibling bands render, with the child inside its band.
    expect(find.text('Band A'), findsWidgets);
    expect(find.text('Band B'), findsWidgets);
    expect(find.text('A leaf'), findsOneWidget);

    // Minimap tap jumps to a band without error.
    await tester.tap(find.byType(RoadmapMinimap));
    await tester.pumpAndSettle();
    expect(find.byType(RoadmapCanvas), findsOneWidget);
  });
}
