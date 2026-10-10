import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/project/project_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// Pins the Evidence page's write flows (edit / delete project, post / edit /
// remove an update, repo save + sync) end to end against a recording fake, so
// restructuring the page cannot silently drop a verb, an argument, or a reload.
class _Api extends ApiClient {
  _Api()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final calls = <String>[];
  var loads = 0;

  @override
  Future<Map<String, dynamic>> getProject(String id) async {
    loads++;
    return {
      'id': id,
      'workspaceId': 'ws1',
      'title': 'Flows',
      'description': 'old desc',
      'status': 'active',
      'repoUrl': '',
      'ownerId': 'u1',
      'updates': [
        {
          'id': 'up1',
          'kind': 'note',
          'content': 'shipped the flow',
          'createdAt': '2026-10-01T00:00:00Z',
          'authorId': 'u1',
        },
      ],
    };
  }

  @override
  Future<List<RoadmapNode>> listRoadmap(String projectId) async => const [];

  @override
  Future<Project> updateProject(
    String id, {
    required String title,
    required String description,
    required String status,
    String? ownerId,
  }) async {
    calls.add('updateProject:$id:$title:$description:$status:$ownerId');
    return Project.fromJson({
      'id': id,
      'workspaceId': 'ws1',
      'title': title,
      'description': description,
      'status': status,
      'repoUrl': '',
      'createdAt': '2026-10-01T00:00:00Z',
      'flagged': false,
    });
  }

  @override
  Future<void> deleteProject(String id) async => calls.add('deleteProject:$id');

  @override
  Future<void> addUpdate(String projectId, String kind, String content) async =>
      calls.add('addUpdate:$projectId:$kind:$content');

  @override
  Future<void> editUpdate(String updateId, String content) async =>
      calls.add('editUpdate:$updateId:$content');

  @override
  Future<void> removeUpdate(String updateId) async =>
      calls.add('removeUpdate:$updateId');

  @override
  Future<Project> setRepo(String projectId, String repoUrl) async {
    calls.add('setRepo:$projectId:$repoUrl');
    return Project.fromJson({
      'id': projectId,
      'workspaceId': 'ws1',
      'title': 'Flows',
      'description': '',
      'status': 'active',
      'repoUrl': repoUrl,
      'createdAt': '2026-10-01T00:00:00Z',
      'flagged': false,
    });
  }

  @override
  Future<Map<String, dynamic>> syncGithub(String projectId) async {
    calls.add('syncGithub:$projectId');
    return {'added': 3};
  }
}

Future<_Api> _pump(WidgetTester tester) async {
  final api = _Api();
  final session = Session()
    ..ready = true
    ..api = api;
  final router = GoRouter(
    routes: [
      GoRoute(
        path: '/',
        builder: (_, _) => const ProjectPage(id: 'p1'),
      ),
      GoRoute(
        path: '/workspace/:id',
        builder: (_, state) => Text('workspace ${state.pathParameters['id']}'),
      ),
      GoRoute(
        path: '/projects/:id/roadmap',
        builder: (_, state) => Text('roadmap ${state.pathParameters['id']}'),
      ),
    ],
  );
  await tester.binding.setSurfaceSize(const Size(1400, 1200));
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
      child: MaterialApp.router(routerConfig: router),
    ),
  );
  await tester.pumpAndSettle();
  return api;
}

Finder _inDialog(Finder finder) =>
    find.descendant(of: find.byType(Dialog), matching: finder);

void main() {
  testWidgets('edit project writes the trimmed title, description and status', (
    tester,
  ) async {
    final api = await _pump(tester);
    expect(api.loads, 1);

    await tester.tap(find.text('Edit project'));
    await tester.pumpAndSettle();
    final fields = _inDialog(find.byType(TextField));
    await tester.enterText(fields.at(0), '  Renamed  ');
    await tester.enterText(fields.at(1), 'new desc');
    await tester.tap(_inDialog(find.byType(DropdownButton<String>)));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Paused').last);
    await tester.pumpAndSettle();
    await tester.tap(_inDialog(find.text('Save')));
    await tester.pumpAndSettle();

    expect(api.calls, ['updateProject:p1:Renamed:new desc:paused:u1']);
    expect(api.loads, 2, reason: 'a saved edit reloads the page');
  });

  testWidgets('cancelling the edit writes nothing', (tester) async {
    final api = await _pump(tester);
    await tester.tap(find.text('Edit project'));
    await tester.pumpAndSettle();
    await tester.tap(_inDialog(find.text('Cancel')));
    await tester.pumpAndSettle();
    expect(api.calls, isEmpty);
    expect(api.loads, 1);
  });

  testWidgets(
    'delete project confirms, deletes, then leaves for the workspace',
    (tester) async {
      final api = await _pump(tester);
      await tester.tap(find.text('Delete'));
      await tester.pumpAndSettle();
      expect(find.text('Delete this project?'), findsOneWidget);
      await tester.tap(_inDialog(find.text('Delete')).last);
      await tester.pumpAndSettle();

      expect(api.calls, ['deleteProject:p1']);
      expect(find.text('workspace ws1'), findsOneWidget);
    },
  );

  testWidgets('post an update sends the kind and the trimmed text', (
    tester,
  ) async {
    final api = await _pump(tester);
    await tester.tap(find.text('Post an update'));
    await tester.pumpAndSettle();
    await tester.tap(_inDialog(find.byType(DropdownButton<String>)));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Blocker').last);
    await tester.pumpAndSettle();
    await tester.enterText(_inDialog(find.byType(TextField)), '  stuck  ');
    await tester.tap(_inDialog(find.text('Post')));
    await tester.pumpAndSettle();

    expect(api.calls, ['addUpdate:p1:blocker:stuck']);
    expect(api.loads, 2);
  });

  testWidgets('an empty update is not posted', (tester) async {
    final api = await _pump(tester);
    await tester.tap(find.text('Post an update'));
    await tester.pumpAndSettle();
    await tester.tap(_inDialog(find.text('Post')));
    await tester.pumpAndSettle();
    expect(api.calls, isEmpty);
    expect(api.loads, 1);
  });

  testWidgets('a note can be edited and removed from its menu', (tester) async {
    final api = await _pump(tester);

    await tester.tap(find.byIcon(Icons.more_vert));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Edit').last);
    await tester.pumpAndSettle();
    await tester.enterText(_inDialog(find.byType(TextField)), 'shipped it');
    await tester.tap(_inDialog(find.text('Save')));
    await tester.pumpAndSettle();
    expect(api.calls, ['editUpdate:up1:shipped it']);
    expect(api.loads, 2);

    await tester.tap(find.byIcon(Icons.more_vert));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Delete').last);
    await tester.pumpAndSettle();
    expect(api.calls, ['editUpdate:up1:shipped it', 'removeUpdate:up1']);
    expect(api.loads, 3);
  });

  testWidgets('repo save and sync go through the GitHub card', (tester) async {
    final api = await _pump(tester);

    await tester.enterText(find.byType(TextField), '  owner/repo  ');
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
    expect(api.calls, ['setRepo:p1:owner/repo']);
    expect(api.loads, 2);

    await tester.tap(find.text('Sync now'));
    await tester.pumpAndSettle();
    expect(api.calls.last, 'syncGithub:p1');
    expect(find.text('Synced — 3 new commit(s) added.'), findsOneWidget);
  });

  testWidgets('the roadmap button opens this project\'s roadmap', (
    tester,
  ) async {
    await _pump(tester);
    await tester.tap(find.text('Roadmap'));
    await tester.pumpAndSettle();
    expect(find.text('roadmap p1'), findsOneWidget);
  });
}
