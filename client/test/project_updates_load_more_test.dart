import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/api/requests.dart';
import 'package:insideout/features/project/project_updates_feed.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// rv5-PARB-07: the feed reads older updates page by page from nextCursor.
class _Api extends ApiClient {
  _Api()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final befores = <String?>[];

  @override
  Future<UpdatesPage> listProjectUpdates(
    String projectId, {
    String? before,
  }) async {
    befores.add(before);
    return (
      updates: [
        ProjectUpdate(
          id: 'u0',
          kind: 'note',
          content: 'The oldest note',
          createdAt: '2026-10-01T00:00:00Z',
          authorId: 'a1',
        ),
      ],
      nextCursor: null,
    );
  }
}

void main() {
  test('the page request names the cursor', () {
    final c = getProjectUpdates('p1', before: 'u1');
    expect(c.method, 'GET');
    expect(c.path, '/projects/p1/updates?before=u1');
  });

  testWidgets('load more appends the next page and stops at the end', (
    tester,
  ) async {
    final api = _Api();
    await tester.pumpWidget(_feed(api, [_newest()]));
    await tester.pumpAndSettle();
    expect(find.text('The oldest note'), findsNothing);

    await tester.tap(find.text('Load older updates'));
    await tester.pumpAndSettle();
    expect(api.befores, ['u1']);
    expect(find.text('The newest note'), findsOneWidget);
    expect(find.text('The oldest note'), findsOneWidget);
    expect(find.text('Load older updates'), findsNothing);
  });

  // The project page builds a new list on every rebuild: an equal first page
  // must not drop the older pages already read.
  testWidgets('a parent rebuild with an equal list keeps the older pages', (
    tester,
  ) async {
    final api = _Api();
    await tester.pumpWidget(_feed(api, [_newest()]));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Load older updates'));
    await tester.pumpAndSettle();
    expect(find.text('The oldest note'), findsOneWidget);

    await tester.pumpWidget(_feed(api, [_newest()]));
    await tester.pumpAndSettle();
    expect(find.text('The oldest note'), findsOneWidget);
    expect(find.text('Load older updates'), findsNothing);
  });

  testWidgets('a new newest update starts the history over', (tester) async {
    final api = _Api();
    await tester.pumpWidget(_feed(api, [_newest()]));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Load older updates'));
    await tester.pumpAndSettle();

    final added = ProjectUpdate(
      id: 'u2',
      kind: 'note',
      content: 'A note just added',
      createdAt: '2026-11-02T00:00:00Z',
      authorId: 'a1',
    );
    await tester.pumpWidget(_feed(api, [added, _newest()]));
    await tester.pumpAndSettle();
    expect(find.text('A note just added'), findsOneWidget);
    expect(find.text('The oldest note'), findsNothing);
    expect(find.text('Load older updates'), findsOneWidget);
  });
}

ProjectUpdate _newest() => ProjectUpdate(
  id: 'u1',
  kind: 'note',
  content: 'The newest note',
  createdAt: '2026-11-01T00:00:00Z',
  authorId: 'a1',
);

Widget _feed(_Api api, List<ProjectUpdate> updates) {
  final session = Session()
    ..ready = true
    ..api = api;
  return MultiProvider(
    providers: [
      ChangeNotifierProvider.value(value: session),
      ChangeNotifierProvider(
        create: (_) =>
            Appearance(store: MemorySettingsStore(), locale: 'en-US'),
      ),
    ],
    child: MaterialApp(
      home: Scaffold(
        body: SingleChildScrollView(
          child: ProjectUpdatesFeed(
            canDecide: true,
            projectId: 'p1',
            nextCursor: 'u1',
            onChanged: () async {},
            onEdit: (u, content) async {},
            onRemove: (u) async {},
            updates: updates,
          ),
        ),
      ),
    ),
  );
}
