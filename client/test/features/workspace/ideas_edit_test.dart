import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/workspace/ideas_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// CQ-02 behavioral coverage for idea edit: it used to be guarded only by a
// substring grep in feature_wiring_test — here the dialog must really send
// the idea id and the edited text to updateIdea. Kept out of
// ideas_page_test.dart so that file stays under ~350 lines.
class _FakeApi extends ApiClient {
  _FakeApi(this.ideas)
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final List<Idea> ideas;
  String? savedId;
  String? savedTitle;
  String? savedContent;

  @override
  Future<List<Idea>> listIdeas(String workspaceId) async => ideas;

  @override
  Future<List<Project>> listProjects(String workspaceId) async => [];

  @override
  Future<Idea> updateIdea(
    String id, {
    required String title,
    required String content,
  }) async {
    savedId = id;
    savedTitle = title;
    savedContent = content;
    final i = ideas.indexWhere((idea) => idea.id == id);
    if (i >= 0) ideas[i] = _idea(id, title);
    return _idea(id, title);
  }
}

Idea _idea(String id, String title) => Idea.fromJson({
  'id': id,
  'workspaceId': 'ws1',
  'authorId': 'u1',
  'title': title,
  'content': 'some content',
  'status': 'inbox',
  'prdId': null,
  'createdAt': '2026-10-01T00:00:00Z',
  'updatedAt': '2026-10-01T00:00:00Z',
});

void main() {
  testWidgets('editing an idea saves the id and the new text', (tester) async {
    final api = _FakeApi([_idea('i1', 'Edit me')]);
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
        child: const MaterialApp(home: IdeasPage(workspaceId: 'ws1')),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byIcon(Icons.more_vert));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Edit'));
    await tester.pumpAndSettle();

    // The dialog is seeded with the idea; its title field is the first one
    // inside the dialog (the capture field stays behind the overlay).
    final dialogTitle = find
        .descendant(of: find.byType(Dialog), matching: find.byType(TextField))
        .first;
    expect(tester.widget<TextField>(dialogTitle).controller!.text, 'Edit me');
    await tester.enterText(dialogTitle, 'Edited title');
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(api.savedId, 'i1');
    expect(api.savedTitle, 'Edited title');
    expect(api.savedContent, 'some content');
    // The poster renders the title uppercased, so match case-insensitively.
    expect(
      find.textContaining(RegExp('edited title', caseSensitive: false)),
      findsOneWidget,
    );
  });
}
