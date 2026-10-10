import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/workspace/settings_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/field_card.dart';
import 'package:provider/provider.dart';

// FULL-02, settings half: a member action reloads _load, which reseeded the
// title/description controllers unconditionally — a role change mid-edit
// silently reverted what the member was typing. The reload must keep fields
// whose text moved off the last loaded value, while untouched fields still
// adopt whatever the server now says.
class _FakeApi extends ApiClient {
  _FakeApi(this.ws, this.members)
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  Workspace ws;
  List<WorkspaceMember> members;

  @override
  Future<Workspace> getWorkspace(String id) async => ws;

  @override
  Future<List<WorkspaceMember>> listMembers(String id) async =>
      List.of(members);

  @override
  Future<void> updateMemberRole(
    String workspaceId,
    String userId,
    String role,
  ) async {
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
}

Workspace _ws({
  String title = 'Settings Test WS',
  String description = 'Original desc',
}) => Workspace.fromJson({
  'id': 'ws1',
  'title': title,
  'description': description,
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

Future<void> _pump(WidgetTester tester, _FakeApi api) async {
  final session = Session()
    ..ready = true
    ..api = api;
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

Future<void> _makeTeammateAdmin(WidgetTester tester) async {
  await tester.ensureVisible(find.text('teammate'));
  await tester.tap(_menuOf('teammate'));
  await tester.pumpAndSettle();
  await tester.tap(find.widgetWithText(PopupMenuItem<String>, 'Admin'));
  await tester.pumpAndSettle();
}

String _textOf(WidgetTester tester, int index) =>
    tester.widget<TextField>(find.byType(TextField).at(index)).controller!.text;

void main() {
  testWidgets('a member action reloading mid-edit keeps the unsaved edits', (
    tester,
  ) async {
    final api = _FakeApi(_ws(), [
      _member('u1', 'me', 'admin'),
      _member('u2', 'teammate', 'member'),
    ]);
    await _pump(tester, api);

    await tester.enterText(find.byType(TextField).at(0), 'My rename');
    await tester.enterText(find.byType(TextField).at(1), 'My desc');
    await _makeTeammateAdmin(tester);

    expect(_textOf(tester, 0), 'My rename');
    expect(_textOf(tester, 1), 'My desc');
  });

  testWidgets('an untouched field still adopts the server value on reload', (
    tester,
  ) async {
    final api = _FakeApi(_ws(), [
      _member('u1', 'me', 'admin'),
      _member('u2', 'teammate', 'member'),
    ]);
    await _pump(tester, api);

    // Another admin renamed the workspace while this member had it open; the
    // untouched field must not be pinned to the stale first load forever.
    api.ws = _ws(title: 'Renamed elsewhere', description: 'Newer desc');
    await _makeTeammateAdmin(tester);

    expect(_textOf(tester, 0), 'Renamed elsewhere');
    expect(_textOf(tester, 1), 'Newer desc');
  });
}
