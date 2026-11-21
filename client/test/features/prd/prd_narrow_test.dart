import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/prd_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:provider/provider.dart';

/// The 390px walk found the PRD page crowded: the title field shared a row
/// with "Next: Request review" and was squeezed to a sliver, and the 52px
/// "PRODUCT REQUIREMENTS" heading broke "REQUIREMENTS" mid-word.
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
  Future<Prd> getPrd(String id) async => Prd(
    id: 'prd-1',
    workspaceId: 'ws-1',
    title: 'In-app changelog widget',
    sections: const {},
    status: 'draft',
    currentRevision: 1,
    updatedAt: '2026-10-05T00:00:00Z',
  );

  @override
  Future<Conversation?> conversationForPrd(String prdId) async => null;

  // The page looks up the viewer's role; keep it off the network.
  @override
  Future<Workspace> getWorkspace(String id) async => Workspace.fromJson({
    'id': id,
    'title': 'Board Test WS',
    'description': '',
    'code': 'CODE12',
    'status': 'active',
    'memberCount': 2,
    'myRole': 'admin',
    'createdAt': '2026-01-01T00:00:00Z',
  });
}

Future<void> _pump(WidgetTester tester, {required Size size}) async {
  final session = Session()
    ..ready = true
    ..api = _FakeApi();
  // setSurfaceSize leaves MediaQuery at 800x600; resize the view itself.
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = size;
  addTearDown(tester.view.reset);

  final router = GoRouter(
    initialLocation: '/prd/prd-1',
    routes: [
      GoRoute(
        path: '/prd/:id',
        builder: (_, s) => PrdPage(id: s.pathParameters['id']!),
      ),
    ],
  );
  addTearDown(router.dispose);

  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'en-US'),
        ),
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: MaterialApp.router(routerConfig: router),
    ),
  );
  await tester.pumpAndSettle();
}

// Below Cobalt.bpNarrow. 420 rather than the walk's 390 only because the test
// font (Ahem, one em per glyph) is wider than the real one, and the stacked
// review button would overflow by a few pixels there.
const _phone = Size(420, 844);

double _headingSize(WidgetTester tester) =>
    tester.widget<Text>(find.text('PRODUCT REQUIREMENTS')).style!.fontSize!;

void main() {
  testWidgets(
    'on a phone the title field keeps the row and the button drops under it',
    (tester) async {
      await _pump(tester, size: _phone);

      final field = tester.getRect(find.byType(TextField).first);
      expect(
        field.width,
        greaterThan(300),
        reason: 'the title field is a sliver',
      );

      final review = tester.getRect(
        find.widgetWithText(BlockButton, 'Next: Request review'),
      );
      expect(review.top, greaterThanOrEqualTo(field.bottom));
      expect(review.left, greaterThanOrEqualTo(0));
      expect(review.right, lessThanOrEqualTo(_phone.width));
    },
  );

  testWidgets('on a phone the heading is smaller than on a wide screen', (
    tester,
  ) async {
    await _pump(tester, size: _phone);
    final phone = _headingSize(tester);

    await _pump(tester, size: const Size(1400, 900));
    expect(phone, lessThan(_headingSize(tester)));
  });

  testWidgets('on a wide screen the title field and the button share a row', (
    tester,
  ) async {
    await _pump(tester, size: const Size(1400, 900));

    final field = tester.getRect(find.byType(TextField).first);
    final review = tester.getRect(
      find.widgetWithText(BlockButton, 'Next: Request review'),
    );
    expect(review.left, greaterThanOrEqualTo(field.right));
    expect(review.center.dy, lessThan(field.bottom));
  });

  // The stacked layout put the PRD column in a fixed 640px box that scrolls on
  // its own inside the page's list. A touch drag starting over the column was
  // taken by the inner list and never moved the page, so the coach below it
  // could be reached only by dragging from the coach's own area.
  testWidgets('on a phone a drag over the PRD column scrolls the page', (
    tester,
  ) async {
    await _pump(tester, size: _phone);

    // Keep dragging up from inside the PRD column, as a reader would.
    for (var i = 0; i < 8; i++) {
      await tester.dragFrom(const Offset(200, 300), const Offset(0, -500));
      await tester.pumpAndSettle();
    }

    // The page itself scrolled: the coach block is pulled up the screen
    // (it sat at y=704 while the column's own list took every drag).
    expect(tester.getTopLeft(find.text('PRD COACH')).dy, lessThan(400));
  });
}
