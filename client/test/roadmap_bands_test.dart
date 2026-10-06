import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_bands.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_theme.dart';
import 'package:provider/provider.dart';

// Bands consumes the server progress payload. An in-progress node with no
// deadline is needsDeadline on the server, so it must not land in Now.
class _FakeApi extends ApiClient {
  _FakeApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );
}

void main() {
  testWidgets('in-progress node without a deadline is not in Now', (
    tester,
  ) async {
    final session = Session();
    session.ready = true;
    session.api = _FakeApi();

    final undeadlined = RoadmapNode(
      id: 'n1',
      projectId: 'p1',
      title: 'Docs pass',
      description: '',
      status: 'in_progress',
      position: 0,
    );
    final progress = RoadmapProgress(
      now: const [],
      needsDeadline: [
        ProgressItem(
          id: undeadlined.id,
          title: undeadlined.title,
          status: undeadlined.status,
        ),
      ],
      next: const [],
      doneCount: 0,
    );

    await tester.binding.setSurfaceSize(const Size(1400, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

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
          theme: cobaltTheme(Cobalt.light, brightness: Brightness.light),
          home: Scaffold(
            body: RoadmapBands(
              projectId: 'p1',
              nodes: [undeadlined],
              progress: progress,
              onReload: _noopReload,
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Docs pass'), findsOneWidget);
    expect(find.text('NOW'), findsOneWidget);

    final title = tester.getTopLeft(find.text('Docs pass'));
    final nowLabel = tester.getTopLeft(find.text('NOW'));
    expect(title.dy, isNot(closeTo(nowLabel.dy, 80)));
  });
}

Future<void> _noopReload() async {}
