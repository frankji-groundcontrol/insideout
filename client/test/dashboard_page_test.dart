import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/dashboard/dashboard_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// Register 4.1 coverage: a failing create or join must surface the
// localized error instead of failing silently, and leave the board
// untouched; a successful one reloads. A failing load shows the localized
// error and its Retry reloads the board, never the raw exception text.
class _FakeApi extends ApiClient {
  _FakeApi(this.workspaces, {this.onCreate, this.onJoin})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final List<Workspace> workspaces;
  void Function(String title)? onCreate;
  void Function(String code)? onJoin;

  /// Thrown by [listWorkspaces] while set; clear it to let a Retry succeed.
  Object? listError;
  var listCalls = 0;

  @override
  Future<List<Workspace>> listWorkspaces() async {
    listCalls++;
    final failure = listError;
    if (failure != null) throw failure;
    return workspaces;
  }

  @override
  Future<Workspace> createWorkspace(String title, String description) async {
    onCreate?.call(title);
    return Workspace.fromJson({
      'id': 'w-new',
      'title': title,
      'description': description,
      'code': 'NEW123',
      'status': 'active',
      'memberCount': 1,
      'myRole': 'admin',
      'createdAt': '2026-10-06T00:00:00Z',
    });
  }

  @override
  Future<Workspace> joinWorkspace(String code) async {
    onJoin?.call(code);
    return Workspace.fromJson({
      'id': 'w-joined',
      'title': 'Joined WS',
      'description': '',
      'code': code,
      'status': 'active',
      'memberCount': 2,
      'myRole': 'member',
      'createdAt': '2026-10-06T00:00:00Z',
    });
  }
}

Workspace _workspace(String id, String title) => Workspace.fromJson({
  'id': id,
  'title': title,
  'description': '',
  'code': 'C$id',
  'status': 'active',
  'memberCount': 1,
  'myRole': 'admin',
  'createdAt': '2026-10-06T00:00:00Z',
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
        ChangeNotifierProvider(
          create: (_) =>
              Appearance(store: MemorySettingsStore(), locale: 'en-US'),
        ),
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: const MaterialApp(home: DashboardPage()),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('failed workspace create shows the error and adds nothing', (
    tester,
  ) async {
    final api = _FakeApi(
      const [],
      onCreate: (_) => throw ApiException('boom', status: 500),
    );
    await _pump(tester, api);

    await tester.tap(find.text('Create'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'Failing WS');
    await tester.tap(find.text('Create').last);
    await tester.pumpAndSettle();

    expect(
      find.text('Save failed — Server hiccup — try again'),
      findsOneWidget,
    );
    expect(find.text('Failing WS'), findsNothing);
  });

  testWidgets('failed workspace join shows the error and adds nothing', (
    tester,
  ) async {
    final api = _FakeApi(
      const [],
      onJoin: (_) => throw ApiException('boom', status: 500),
    );
    await _pump(tester, api);

    await tester.tap(find.text('Join'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'BADCODE');
    await tester.tap(find.text('Join').last);
    await tester.pumpAndSettle();

    expect(
      find.text('Save failed — Server hiccup — try again'),
      findsOneWidget,
    );
    expect(find.text('Joined WS'), findsNothing);
  });

  testWidgets('successful create reloads the board with the new workspace', (
    tester,
  ) async {
    final workspaces = <Workspace>[];
    final api = _FakeApi(
      workspaces,
      onCreate: (title) {
        workspaces.add(
          Workspace.fromJson({
            'id': 'w-new',
            'title': title,
            'description': '',
            'code': 'NEW123',
            'status': 'active',
            'memberCount': 1,
            'myRole': 'admin',
            'createdAt': '2026-10-06T00:00:00Z',
          }),
        );
      },
    );
    await _pump(tester, api);

    await tester.tap(find.text('Create'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'Fresh WS');
    await tester.tap(find.text('Create').last);
    await tester.pumpAndSettle();

    expect(find.text('Fresh WS'), findsOneWidget);
  });

  // The card read "1 members" for a workspace of one.
  testWidgets('the member count agrees with its number', (tester) async {
    Workspace ws(String id, int members) => Workspace.fromJson({
      'id': id,
      'title': 'WS $id',
      'description': '',
      'code': 'C$id',
      'status': 'active',
      'memberCount': members,
      'myRole': 'admin',
      'createdAt': '2026-10-06T00:00:00Z',
    });
    await _pump(tester, _FakeApi([ws('a', 1), ws('b', 2)]));

    expect(find.textContaining('· 1 member'), findsOneWidget);
    expect(find.textContaining('1 members'), findsNothing);
    expect(find.textContaining('· 2 members'), findsOneWidget);
  });

  // The create button only existed in the empty state, so once a user had one
  // workspace the board offered join and nothing else.
  testWidgets('a user who already has a workspace can still create another', (
    tester,
  ) async {
    final workspaces = [_workspace('w1', 'First WS')];
    final api = _FakeApi(
      workspaces,
      onCreate: (title) => workspaces.add(_workspace('w2', title)),
    );
    await _pump(tester, api);

    expect(find.text('Create a workspace'), findsOneWidget);
    await tester.tap(find.text('Create a workspace'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'Second WS');
    await tester.tap(find.text('Create').last);
    await tester.pumpAndSettle();

    expect(find.text('Second WS'), findsOneWidget);
    expect(find.text('First WS'), findsOneWidget);
  });

  testWidgets('a failed load shows the localized error, never the exception', (
    tester,
  ) async {
    final api = _FakeApi(const [])
      ..listError = ApiException('boom', status: 500);
    await _pump(tester, api);

    expect(find.text('Something went wrong'), findsOneWidget);
    expect(find.text('Server hiccup — try again'), findsOneWidget);
    expect(find.textContaining('boom'), findsNothing);
  });

  testWidgets('Retry after a failed load reloads the board', (tester) async {
    final api = _FakeApi([_workspace('w1', 'First WS')])
      ..listError = ApiException('boom', status: 500);
    await _pump(tester, api);
    expect(find.text('First WS'), findsNothing);

    api.listError = null;
    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();

    expect(find.text('First WS'), findsOneWidget);
    expect(api.listCalls, 2);
  });
}
