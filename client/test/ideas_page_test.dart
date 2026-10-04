import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/workspace/ideas_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// USER-09 behavioral coverage: drop is terminal, so it must confirm with
// the translated prompt, and cancelling must leave the card untouched.
class _FakeApi extends ApiClient {
  _FakeApi(this.ideas, {this.onDrop}) : super(
            baseUrl: 'http://127.0.0.1:9/api/v1',
            readAccess: () async => null,
            readRefresh: () async => null,
            writeTokens: (a, r) async {},
            clearTokens: () async {},
          );

  final List<Idea> ideas;
  void Function(String id)? onDrop;

  @override
  Future<List<Idea>> listIdeas(String workspaceId) async => ideas;

  @override
  Future<List<Project>> listProjects(String workspaceId) async => [];

  @override
  Future<void> dropIdea(String id) async => onDrop?.call(id);
}

Finder _card(String title) =>
    find.textContaining(RegExp(title, caseSensitive: false));

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
        ChangeNotifierProvider(create: (_) => Appearance(store: MemorySettingsStore(), locale: 'en-US')),
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: const MaterialApp(home: IdeasPage(workspaceId: 'ws1')),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('drop asks for confirmation and cancels cleanly', (tester) async {
    var dropped = false;
    final api = _FakeApi([_idea('i1', 'Keep me')], onDrop: (_) => dropped = true);
    await _pump(tester, api);
    expect(_card('keep me'), findsOneWidget);

    await tester.tap(find.byIcon(Icons.more_vert));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Drop'));
    await tester.pumpAndSettle();

    expect(find.text('Drop this idea?'), findsOneWidget);
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(dropped, isFalse);
    expect(_card('keep me'), findsOneWidget);
  });

  testWidgets('confirming drop calls the API and drops the card', (tester) async {
    final ideas = [_idea('i1', 'Drop me')];
    final api = _FakeApi(ideas, onDrop: (id) => ideas.removeWhere((i) => i.id == id));
    await _pump(tester, api);
    expect(_card('drop me'), findsOneWidget);

    await tester.tap(find.byIcon(Icons.more_vert));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Drop'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Drop').last);
    await tester.pumpAndSettle();

    expect(_card('drop me'), findsNothing);
  });
}
