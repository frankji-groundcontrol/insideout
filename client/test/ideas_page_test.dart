import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/errors.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/workspace/ideas_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:provider/provider.dart';

// USER-09 behavioral coverage: drop is terminal, so it must confirm with
// the translated prompt, and cancelling must leave the card untouched.
class _FakeApi extends ApiClient {
  _FakeApi(this.ideas, {this.onDrop, this.onCreate, this.onConvert})
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  final List<Idea> ideas;
  void Function(String id)? onDrop;
  void Function(String title)? onCreate;
  Future<Map<String, String>> Function(String id)? onConvert;

  /// While set, every write waits on it, so a test can unmount the page with
  /// the write still in flight.
  Completer<void>? gate;

  @override
  Future<List<Idea>> listIdeas(String workspaceId) async => ideas;

  @override
  Future<List<Project>> listProjects(String workspaceId) async => [];

  @override
  Future<Idea> createIdea(
    String workspaceId,
    String title,
    String content,
  ) async {
    await gate?.future;
    onCreate?.call(title);
    final idea = _idea('i-${ideas.length + 1}', title);
    ideas.add(idea);
    return idea;
  }

  @override
  Future<void> dropIdea(String id) async {
    await gate?.future;
    onDrop?.call(id);
  }

  @override
  Future<Idea> updateIdea(
    String id, {
    required String title,
    required String content,
  }) async {
    await gate?.future;
    return _idea(id, title);
  }

  @override
  Future<Map<String, String>> convertIdea(String ideaId) async {
    final convert = onConvert;
    if (convert != null) return convert(ideaId);
    return {'prdId': 'prd-1'};
  }
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
}

// FS-08: a write that outlives its page must not touch the disposed capture
// field or call setState on the dead state. Unmount, then let the write finish.
Future<void> _unmountThenFinish(
  WidgetTester tester,
  Completer<void> gate,
) async {
  await tester.pumpWidget(const SizedBox());
  gate.complete();
  await tester.pump();
  expect(tester.takeException(), isNull);
}

void main() {
  testWidgets('drop asks for confirmation and cancels cleanly', (tester) async {
    var dropped = false;
    final api = _FakeApi([
      _idea('i1', 'Keep me'),
    ], onDrop: (_) => dropped = true);
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

  testWidgets('confirming drop calls the API and drops the card', (
    tester,
  ) async {
    final ideas = [_idea('i1', 'Drop me')];
    final api = _FakeApi(
      ideas,
      onDrop: (id) => ideas.removeWhere((i) => i.id == id),
    );
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

  // 4.2: a failed capture must not eat the user's text — the error shows
  // and the field keeps its content for a retry.
  testWidgets('failed capture keeps the typed text and shows the error', (
    tester,
  ) async {
    final api = _FakeApi([
      _idea('i1', 'Keep me'),
    ], onCreate: (_) => throw ApiException('boom', status: 500));
    await _pump(tester, api);

    await tester.enterText(find.byType(TextField), 'Ship the cobalt wave');
    await tester.tap(find.text('Capture'));
    await tester.pumpAndSettle();

    expect(
      find.text('Save failed — Server hiccup — try again'),
      findsOneWidget,
    );
    expect(
      tester.widget<TextField>(find.byType(TextField)).controller!.text,
      'Ship the cobalt wave',
    );
  });

  testWidgets('successful capture clears the field', (tester) async {
    final api = _FakeApi([_idea('i1', 'Keep me')]);
    await _pump(tester, api);

    await tester.enterText(find.byType(TextField), 'Ship the cobalt wave');
    await tester.tap(find.text('Capture'));
    await tester.pumpAndSettle();

    expect(
      tester.widget<TextField>(find.byType(TextField)).controller!.text,
      isEmpty,
    );
    expect(_card('ship the cobalt wave'), findsOneWidget);
  });

  testWidgets('failed convert shows the error and leaves the card', (
    tester,
  ) async {
    final api = _FakeApi([
      _idea('i1', 'Keep me'),
    ], onConvert: (_) async => throw ApiException('boom', status: 500));
    await _pump(tester, api);

    await tester.tap(find.text('Convert to PRD'));
    await tester.pumpAndSettle();

    expect(
      find.text('Save failed — Server hiccup — try again'),
      findsOneWidget,
    );
    expect(_card('keep me'), findsOneWidget);
  });

  // The poster looks like the primary object, so its body does what its
  // call-to-action does (here: convert, which fails and says so).
  testWidgets('tapping the poster body runs the same action as its button', (
    tester,
  ) async {
    final api = _FakeApi([
      _idea('i1', 'Keep me'),
    ], onConvert: (_) async => throw ApiException('boom', status: 500));
    await _pump(tester, api);

    await tester.tap(_card('keep me'));
    await tester.pumpAndSettle();

    expect(
      find.text('Save failed — Server hiccup — try again'),
      findsOneWidget,
    );
    expect(_card('keep me'), findsOneWidget);
  });

  // The whole-card tap must not swallow the controls inside it.
  testWidgets('the more-actions menu opens without running the card action', (
    tester,
  ) async {
    var converts = 0;
    final api = _FakeApi(
      [_idea('i1', 'Keep me')],
      onConvert: (_) async {
        converts++;
        throw ApiException('boom', status: 500);
      },
    );
    await _pump(tester, api);

    await tester.tap(find.byIcon(Icons.more_vert));
    await tester.pumpAndSettle();

    expect(find.text('Drop'), findsOneWidget);
    expect(converts, 0);
  });

  testWidgets('a capture that finishes after the page is gone does not throw', (
    tester,
  ) async {
    final gate = Completer<void>();
    await _pump(tester, _FakeApi([_idea('i1', 'Keep me')])..gate = gate);

    await tester.enterText(find.byType(TextField), 'Ship the cobalt wave');
    await tester.tap(find.text('Capture'));
    await tester.pump();

    await _unmountThenFinish(tester, gate);
  });

  testWidgets('a drop that finishes after the page is gone does not throw', (
    tester,
  ) async {
    final gate = Completer<void>();
    await _pump(tester, _FakeApi([_idea('i1', 'Drop me')])..gate = gate);

    await tester.tap(find.byIcon(Icons.more_vert));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Drop'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Drop').last);
    await tester.pump();

    await _unmountThenFinish(tester, gate);
  });

  testWidgets('an edit that finishes after the page is gone does not throw', (
    tester,
  ) async {
    final gate = Completer<void>();
    await _pump(tester, _FakeApi([_idea('i1', 'Edit me')])..gate = gate);

    await tester.tap(find.byIcon(Icons.more_vert));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Edit'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Save'));
    await tester.pump();

    await _unmountThenFinish(tester, gate);
  });
}
