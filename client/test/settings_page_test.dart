import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/workspace/settings_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/field_card.dart';
import 'package:provider/provider.dart';

// CQ-02 behavioral coverage for the settings verbs that used to be guarded
// only by substring greps: workspace rename, member role change, member
// removal — each asserted through what the page actually sends to the API.
class _FakeApi extends ApiClient {
  _FakeApi(this.ws, this.members, {this.failWrites = false})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  Workspace ws;
  List<WorkspaceMember> members;
  final bool failWrites;

  /// Every mutation the page makes it to the API, in order.
  final List<String> calls = [];

  @override
  Future<Workspace> getWorkspace(String id) async => ws;

  @override
  Future<List<WorkspaceMember>> listMembers(String id) async =>
      List.of(members);

  @override
  Future<Workspace> updateWorkspace(
    String id, {
    required String title,
    required String description,
    String? coverUrl,
  }) async {
    if (failWrites) throw ApiException('boom', status: 500);
    calls.add('updateWorkspace($id, title: $title, description: $description)');
    ws = Workspace(
      id: ws.id,
      title: title,
      description: description,
      code: ws.code,
      status: ws.status,
      memberCount: ws.memberCount,
      myRole: ws.myRole,
      createdAt: ws.createdAt,
    );
    return ws;
  }

  @override
  Future<void> updateMemberRole(
    String workspaceId,
    String userId,
    String role,
  ) async {
    if (failWrites) throw ApiException('boom', status: 500);
    calls.add('updateMemberRole($workspaceId, $userId, $role)');
    final i = members.indexWhere((m) => m.userId == userId);
    if (i >= 0) {
      final m = members[i];
      members[i] = WorkspaceMember(
        userId: m.userId,
        username: m.username,
        email: m.email,
        role: role,
      );
    }
  }

  @override
  Future<void> removeMember(String workspaceId, String userId) async {
    if (failWrites) throw ApiException('boom', status: 500);
    calls.add('removeMember($workspaceId, $userId)');
    members.removeWhere((m) => m.userId == userId);
  }
}

Workspace _ws() => Workspace.fromJson({
  'id': 'ws1',
  'title': 'Settings Test WS',
  'description': 'Original desc',
  'code': 'CODE12',
  'status': 'active',
  'memberCount': 2,
  'myRole': 'admin',
  'createdAt': '2026-01-01T00:00:00Z',
});

WorkspaceMember _member(String userId, String username, String role) =>
    WorkspaceMember.fromJson({
      'userId': userId,
      'username': username,
      'email': '$username@example.net',
      'role': role,
    });

List<WorkspaceMember> _pair() => [
  _member('u1', 'me', 'admin'),
  _member('u2', 'teammate', 'member'),
];

Future<void> _pump(WidgetTester tester, _FakeApi api) async {
  final session = Session();
  session.ready = true;
  session.api = api;
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
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: const MaterialApp(home: SettingsPage(workspaceId: 'ws1')),
    ),
  );
  await tester.pumpAndSettle();
}

/// The "More" menu on one member's card (each card has one).
Finder _menuOf(String username) {
  final card = find.ancestor(
    of: find.text(username),
    matching: find.byType(FieldCard),
  );
  return find.descendant(of: card, matching: find.byTooltip('More'));
}

void main() {
  testWidgets('saving the form renames the workspace through the API', (
    tester,
  ) async {
    final api = _FakeApi(_ws(), _pair());
    await _pump(tester, api);

    // The form is seeded from the workspace; the name field is first.
    expect(
      tester.widget<TextField>(find.byType(TextField).first).controller!.text,
      'Settings Test WS',
    );
    await tester.enterText(find.byType(TextField).first, 'Renamed WS');
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(api.calls, [
      'updateWorkspace(ws1, title: Renamed WS, description: Original desc)',
    ]);
  });

  testWidgets(
    'changing a member role calls updateMemberRole and relabels the row',
    (tester) async {
      final api = _FakeApi(_ws(), _pair());
      await _pump(tester, api);

      await tester.ensureVisible(find.text('teammate'));
      await tester.tap(_menuOf('teammate'));
      await tester.pumpAndSettle();
      await tester.tap(find.widgetWithText(PopupMenuItem<String>, 'Admin'));
      await tester.pumpAndSettle();

      expect(api.calls, ['updateMemberRole(ws1, u2, admin)']);
      expect(find.text('teammate@example.net · Member'), findsNothing);
      expect(find.text('teammate@example.net · Admin'), findsOneWidget);
    },
  );

  testWidgets('removing a member calls removeMember and drops the row', (
    tester,
  ) async {
    final api = _FakeApi(_ws(), _pair());
    await _pump(tester, api);

    await tester.ensureVisible(find.text('teammate'));
    await tester.tap(_menuOf('teammate'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(PopupMenuItem<String>, 'Remove'));
    await tester.pumpAndSettle();

    expect(api.calls, ['removeMember(ws1, u2)']);
    expect(find.text('teammate'), findsNothing);
    expect(find.text('me'), findsOneWidget);
  });

  // Register 4.1 pattern: a failing write must surface the localized banner,
  // keep the typed input, and swallow the exception (no unhandled error).
  testWidgets('a failing save shows the localized banner and keeps the input', (
    tester,
  ) async {
    final api = _FakeApi(_ws(), _pair(), failWrites: true);
    await _pump(tester, api);

    await tester.enterText(find.byType(TextField).first, 'Doomed rename');
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(
      find.text('Save failed — Server hiccup — try again'),
      findsOneWidget,
    );
    expect(
      tester.widget<TextField>(find.byType(TextField).first).controller!.text,
      'Doomed rename',
    );
    expect(tester.takeException(), isNull);
  });
}
