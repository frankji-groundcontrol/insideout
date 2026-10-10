import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/project_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:provider/provider.dart';

// The shipped default locale is zh-CN, yet the project (evidence) page was
// only ever widget-tested in en-US (project_page_test.dart). Pump it in
// zh-CN at phone width and hold it to the strings the zh table carries:
// lookup silently falls back to English (l10n.dart), so a missing key must
// fail here, not in front of a Chinese reader.
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
    'title': '证据页中文走查',
    'description': '',
    'status': 'active',
    'repoUrl': '',
    'ownerId': 'u1',
    'updates': [],
  };

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => const [];
}

// 420 rather than 390 because the Ahem test font is wider; both sit below
// Cobalt.bpNarrow (docs/learning/2026-10-10-test-the-width-you-ship-for.md).
const _phone = Size(420, 844);

Future<LoopFocus> _pump(WidgetTester tester) async {
  final session = Session()
    ..ready = true
    ..api = _FakeApi();
  final focus = LoopFocus();
  // setSurfaceSize leaves MediaQuery at 800x600; resize the view itself.
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = _phone;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'zh-CN'),
        ),
        ChangeNotifierProvider.value(value: focus),
      ],
      child: const MaterialApp(home: ProjectPage(id: 'p1')),
    ),
  );
  await tester.pumpAndSettle();
  return focus;
}

void _expectOnScreen(Rect rect) {
  expect(rect.left, greaterThanOrEqualTo(0), reason: '$rect leaves the left');
  expect(rect.top, greaterThanOrEqualTo(0), reason: '$rect leaves the top');
  expect(rect.right, lessThanOrEqualTo(_phone.width), reason: '$rect: right');
  expect(
    rect.bottom,
    lessThanOrEqualTo(_phone.height),
    reason: '$rect: bottom',
  );
}

void main() {
  testWidgets('the project page speaks zh-CN and stays on the phone', (
    tester,
  ) async {
    await _pump(tester);
    expect(tester.takeException(), isNull);

    // Distinctive zh strings; an English fallback would fail these.
    expect(find.text('证据页中文走查'), findsOneWidget);
    expect(find.text('证据'), findsOneWidget); // evidence.title eyebrow
    expect(
      find.text('Git 与项目更新落在同一块场地上。'), // evidence.lead
      findsOneWidget,
    );
    expect(find.textContaining('关联一个 GitHub 仓库'), findsOneWidget);

    // The roadmap jump rides the header's action wrap.
    _expectOnScreen(tester.getRect(find.widgetWithText(BlockButton, '路线图')));

    // The primary action is the page-wide add-update button at the bottom.
    // The fields inside the page carry Scrollables of their own, so scroll
    // the outer list by hand (the sibling tests drag it the same way).
    final addButton = find.widgetWithText(BlockButton, '发布进展');
    for (var i = 0; i < 20 && addButton.evaluate().isEmpty; i++) {
      await tester.drag(find.byType(ListView).first, const Offset(0, -400));
      await tester.pumpAndSettle();
    }
    await tester.ensureVisible(addButton);
    await tester.pumpAndSettle();
    _expectOnScreen(tester.getRect(addButton));
    expect(tester.takeException(), isNull);
  });
}
