import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/project/roadmap_progress_view.dart';
import 'package:insideout/theme/block_button.dart';
import 'package:insideout/theme/cobalt.dart';

import '_roadmap_canvas_helpers.dart';
import '_roadmap_tree_helpers.dart';

// The full roadmap in node windows (2026-11-22): a node's window is the
// node's own roadmap page, embedded, laid out to the window's size.

/// TreeApi with Priya in charge, Refunds due Now, and each presence stream's
/// stop signal kept, one per page that opened one.
class _EmbedApi extends TreeApi {
  _EmbedApi({super.canWrite, super.canWritePlan}) {
    glance = {
      'inCharge': {'userId': 'u-dana', 'name': 'Priya Nair'},
      'now': [
        {'id': 'a1', 'title': 'Refunds', 'status': 'in_progress'},
      ],
    };
  }

  final presenceStops = <Future<void>>[];

  @override
  Future<RoadmapProgress> roadmapProgress(String projectId) async =>
      RoadmapProgress(
        now: [
          ProgressItem.fromJson({
            'id': 'a1',
            'title': 'Refunds',
            'status': 'in_progress',
            'deadline': '2026-12-01',
            'pressure': 'normal',
          }),
        ],
        needsDeadline: [],
        next: [],
        doneCount: 0,
      );

  @override
  Future<void> presenceStream(
    String projectId,
    String clientId,
    void Function(List<PresenceEntry> entries) onChange, {
    void Function(String sessionId, String name, double x, double y)? onCursor,
    Future<void>? stop,
  }) {
    presenceStops.add(stop!);
    return super.presenceStream(projectId, clientId, onChange, stop: stop);
  }
}

/// The page on Canvas with Payments' window open.
Future<GoRouter> _openPayments(WidgetTester tester, [_EmbedApi? api]) async {
  final router = await pumpTreeRouter(tester, treeSession(api ?? _EmbedApi()));
  router.go('/projects/p1/roadmap?view=canvas');
  await tester.pumpAndSettle();
  await tapNode(tester, stepBox('a'));
  return router;
}

Finder _window(String id) => find.byKey(Key('roadmap-window-$id'));
Finder _in(String id, Finder f) =>
    find.descendant(of: _window(id), matching: f);
String _path(GoRouter r) => r.routerDelegate.currentConfiguration.uri.path;

/// Refunds on the Progress view inside [id]'s window.
Finder _refunds(String id) => find.descendant(
  of: _in(id, find.byType(RoadmapProgressView)),
  matching: find.text('Refunds'),
);

/// Shows each view inside [id]'s window in turn.
Future<void> _everyView(WidgetTester tester, String id) async {
  for (var i = 0; i < 4; i++) {
    final choice = _in(id, find.byType(BlockChoice)).at(i);
    tester.widget<BlockChoice>(choice).onTap!();
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  }
}

void main() {
  testWidgets('a window opens on Progress; its Canvas shows the steps', (
    tester,
  ) async {
    await _openPayments(tester);
    expect(_in('a', find.byType(RoadmapProgressView)), findsOneWidget);
    for (final view in ['Progress', 'Bands', 'Canvas', "Priya's"]) {
      expect(_in('a', find.text(view)), findsOneWidget);
    }
    await tester.tap(_in('a', find.text('Canvas')));
    await tester.pumpAndSettle();
    expect(_in('a', stepBox('a1')), findsOneWidget);
  });

  testWidgets('the person-in-charge banner is inside the window', (
    tester,
  ) async {
    await _openPayments(tester);
    expect(_in('a', find.byKey(const Key('roadmap-lens'))), findsOneWidget);
    expect(_in('a', find.textContaining("PRIYA NAIR'S VIEW")), findsOneWidget);
  });

  testWidgets('New item shows in a window for a writer only', (tester) async {
    await _openPayments(tester);
    expect(_in('a', find.text('New item')), findsOneWidget);

    await _openPayments(
      tester,
      _EmbedApi(canWrite: false, canWritePlan: false),
    );
    expect(_in('a', find.text('New item')), findsNothing);
  });

  testWidgets('a tap inside a window opens another window', (tester) async {
    final router = await _openPayments(tester);
    await tapNode(tester, _refunds('a'));
    expect(_window('a'), findsOneWidget);
    expect(_window('a1'), findsOneWidget);
    expect(_path(router), '/projects/p1/roadmap');
  });

  testWidgets('a double-click inside a window opens the node as the page', (
    tester,
  ) async {
    final router = await _openPayments(tester);
    await doubleTapNode(tester, _refunds('a'));
    expect(_path(router), '/projects/p1/roadmap/a1');
    expect(_window('a'), findsNothing);
  });

  testWidgets('a breadcrumb inside a window re-roots the window', (
    tester,
  ) async {
    final router = await _openPayments(tester);
    await tester.tap(
      find.descendant(
        of: _in('a', find.byKey(const Key('roadmap-breadcrumb'))),
        matching: find.text('Ship'),
      ),
    );
    await tester.pumpAndSettle();

    expect(_path(router), '/projects/p1/roadmap');
    expect(_window('a'), findsNothing);
    expect(_window('r'), findsOneWidget);
    expect(
      tester.widget<Text>(find.byKey(const Key('roadmap-window-title-r'))).data,
      'Ship',
    );
  });

  testWidgets('closing a window stops its live stream, not the page\'s', (
    tester,
  ) async {
    final api = _EmbedApi();
    await _openPayments(tester, api);
    expect(api.presenceStops, hasLength(2), reason: 'the page, the window');
    final stopped = <int>{};
    for (final (i, stop) in api.presenceStops.indexed) {
      stop.then((_) => stopped.add(i));
    }
    await tester.pump();
    expect(stopped, isEmpty);

    await tester.tap(find.byKey(const Key('roadmap-window-close-a')));
    await tester.pumpAndSettle();
    expect(stopped, {1});
  });

  testWidgets('no view overflows a default or a minimum window', (
    tester,
  ) async {
    await _openPayments(tester);
    expect(tester.getSize(_window('a')), const Size(760, 600));
    await _everyView(tester, 'a');

    await tester.drag(
      find.byKey(const Key('roadmap-window-resize-corner')),
      const Offset(-1000, -1000),
    );
    await tester.pumpAndSettle();
    expect(tester.getSize(_window('a')), const Size(320, 240));
    await _everyView(tester, 'a');
  });

  testWidgets('a window leaves no room for a dock it does not have', (
    tester,
  ) async {
    await _openPayments(tester);
    final list = tester.widget<ListView>(
      find
          .descendant(
            of: _in('a', find.byType(RoadmapProgressView)),
            matching: find.byType(ListView),
          )
          .first,
    );
    expect((list.padding! as EdgeInsets).bottom, 16);
  });

  testWidgets('the page in a window reads the window as its screen', (
    tester,
  ) async {
    await _openPayments(tester);
    MediaQueryData query() => MediaQuery.of(
      tester.element(_in('a', find.byType(RoadmapProgressView))),
    );
    expect(query().size.width, 760);
    expect(query().size.height, lessThan(600));
    expect(query().padding, EdgeInsets.zero);

    await tester.drag(
      find.byKey(const Key('roadmap-window-resize-corner')),
      const Offset(-300, 0),
    );
    await tester.pumpAndSettle();
    expect(query().size.width, 460);
    expect(query().size.width, lessThan(Cobalt.bpNarrow));
  });

  testWidgets('a tap inside a phone sheet opens the next sheet', (
    tester,
  ) async {
    final router = await pumpTreeRouter(tester, treeSession(_EmbedApi()));
    tester.view
      ..devicePixelRatio = 1
      ..physicalSize = const Size(400, 800);
    addTearDown(tester.view.reset);
    await tester.binding.setSurfaceSize(const Size(400, 800));
    router.go('/projects/p1/roadmap');
    await tester.pumpAndSettle();
    await tapNode(tester, find.text('Refunds'));
    expect(find.byType(Dialog), findsOneWidget);

    await tapNode(tester, _refunds('a1'));
    expect(find.byType(Dialog), findsNWidgets(2));
  });
}
