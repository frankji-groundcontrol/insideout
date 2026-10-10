import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/prd/prd_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// The status menu offered Approved and Rejected to every viewer, but the server
// answers 403 to anyone but a workspace admin (store/prds.go UpdatePrdStatus),
// so a plain member's click could only fail. The menu reads the viewer's role
// from the workspace and, while that is unknown, keeps offering both: the
// server still decides.
class _RoleApi extends ApiClient {
  _RoleApi(this.role)
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  /// The viewer's role in the workspace; null makes the lookup fail.
  final String? role;

  @override
  Future<Prd> getPrd(String id) async => Prd(
    id: id,
    workspaceId: 'ws-1',
    title: 'Ship a thing',
    sections: const {},
    status: 'reviewing',
    currentRevision: 1,
    updatedAt: '2026-10-05T00:00:00Z',
  );

  @override
  Future<Conversation?> conversationForPrd(String prdId) async => null;

  @override
  Future<Workspace> getWorkspace(String id) async {
    final role = this.role;
    if (role == null) throw ApiException('workspace failed', status: 500);
    return Workspace.fromJson({
      'id': id,
      'title': 'Board Test WS',
      'description': '',
      'code': 'CODE12',
      'status': 'active',
      'memberCount': 2,
      'myRole': role,
      'createdAt': '2026-01-01T00:00:00Z',
    });
  }
}

Future<void> _openMenu(WidgetTester tester, ApiClient api) async {
  final session = Session()
    ..ready = true
    ..api = api;
  await tester.binding.setSurfaceSize(const Size(1400, 900));
  addTearDown(() => tester.binding.setSurfaceSize(null));
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
  // The page loads first; the role lookup lands a moment after it.
  await tester.pumpAndSettle();
  await tester.tap(find.byTooltip('More'));
  await tester.pumpAndSettle();
}

bool _enabled(WidgetTester tester, String label) => tester
    .widget<PopupMenuItem<String>>(
      find.widgetWithText(PopupMenuItem<String>, label),
    )
    .enabled;

void main() {
  testWidgets('a plain member is not offered Approved or Rejected', (
    tester,
  ) async {
    await _openMenu(tester, _RoleApi('member'));

    expect(_enabled(tester, 'Approved'), isFalse);
    expect(_enabled(tester, 'Rejected'), isFalse);
  });

  testWidgets('a workspace admin can approve or reject a PRD in review', (
    tester,
  ) async {
    await _openMenu(tester, _RoleApi('admin'));

    expect(_enabled(tester, 'Approved'), isTrue);
    expect(_enabled(tester, 'Rejected'), isTrue);
  });

  testWidgets('while the role is unknown the menu stays as it was', (
    tester,
  ) async {
    await _openMenu(tester, _RoleApi(null));

    expect(_enabled(tester, 'Approved'), isTrue);
    expect(_enabled(tester, 'Rejected'), isTrue);
  });
}
