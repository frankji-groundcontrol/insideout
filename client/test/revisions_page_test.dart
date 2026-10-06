import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/prd/revisions_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// Register 4.1 coverage: a failing snapshot must surface the localized
// error instead of failing silently, and the revision list must not
// change; a successful snapshot reloads the list.
class _FakeApi extends ApiClient {
  _FakeApi(this.revisions, {this.onCreate})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final List<PrdRevision> revisions;
  void Function(String? note)? onCreate;

  @override
  Future<List<PrdRevision>> listRevisions(String prdId) async => revisions;

  @override
  Future<void> createRevision(String id, {String? note}) async =>
      onCreate?.call(note);
}

PrdRevision _revision(int n) => PrdRevision(
  id: 'rev-$n',
  revision: n,
  createdAt: '2026-10-0${n}T00:00:00Z',
  note: 'snapshot $n',
  sections: const {'problem': 'text'},
);

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
      ],
      child: const MaterialApp(home: RevisionsPage(prdId: 'prd-1')),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('failed snapshot shows the error and keeps the list', (
    tester,
  ) async {
    final api = _FakeApi(
      [_revision(1)],
      onCreate: (_) {
        throw ApiException('boom', status: 500);
      },
    );
    await _pump(tester, api);
    expect(find.byType(ListTile), findsOneWidget);

    await tester.tap(find.byType(FloatingActionButton));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'should not stick');
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(
      find.text('Save failed — Server hiccup — try again'),
      findsOneWidget,
    );
    expect(find.byType(ListTile), findsOneWidget);
    expect(find.text('snapshot 2'), findsNothing);
  });

  testWidgets('successful snapshot reloads the revision list', (tester) async {
    final revisions = [_revision(1)];
    final api = _FakeApi(
      revisions,
      onCreate: (note) {
        revisions.add(
          PrdRevision(
            id: 'rev-2',
            revision: 2,
            createdAt: '2026-10-06T00:00:00Z',
            note: note,
            sections: const {'problem': 'text'},
          ),
        );
      },
    );
    await _pump(tester, api);

    await tester.tap(find.byType(FloatingActionButton));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'second snapshot');
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(find.byType(ListTile), findsNWidgets(2));
    expect(find.text('second snapshot'), findsOneWidget);
  });
}
