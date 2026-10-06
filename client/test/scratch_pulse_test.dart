import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/project_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

class _FakeApi extends ApiClient {
  _FakeApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  @override
  Future<Map<String, dynamic>> getProject(String id) async => {
    'id': id,
    'workspaceId': 'ws1',
    'title': 'T',
    'description': '',
    'status': 'active',
    'repoUrl': '',
    'ownerId': 'u1',
    'updates': <Map<String, dynamic>>[],
  };

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => const [];
}

void main() {
  testWidgets('pulse debug', (tester) async {
    final session = Session()
      ..ready = true
      ..api = _FakeApi();
    final focus = LoopFocus();
    var listenerFired = 0;
    focus.addListener(() => listenerFired++);
    await tester.binding.setSurfaceSize(const Size(800, 500));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: session),
          ChangeNotifierProvider(
            create: (_) =>
                Appearance(store: MemorySettingsStore(), locale: 'en-US'),
          ),
          ChangeNotifierProvider.value(value: focus),
        ],
        child: const MaterialApp(home: ProjectPage(id: 'p1')),
      ),
    );
    await tester.pumpAndSettle();
    debugPrint(
      'header present: ${find.byKey(const ValueKey("evidence-pulse-flash")).evaluate().length}',
    );
    await tester.drag(find.byType(ListView), const Offset(0, -400));
    await tester.pumpAndSettle();
    final pos = tester
        .state<ScrollableState>(find.byType(Scrollable).first)
        .position;
    debugPrint('after drag: ${pos.pixels}');
    focus.pulseEvidence();
    debugPrint('listener fired total: $listenerFired (incl. page listener)');
    await tester.pump(const Duration(milliseconds: 400));
    debugPrint('after pulse+400ms: ${pos.pixels}');
    debugPrint(
      'header present now: ${find.byKey(const ValueKey("evidence-pulse-flash")).evaluate().length}',
    );
  });
}
