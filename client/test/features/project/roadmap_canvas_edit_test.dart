import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';

import '_roadmap_canvas_helpers.dart';
import '_roadmap_tree_helpers.dart';

// Adding and linking steps on the Canvas (2026-11-22): a "+ Next step" slot,
// "Add a step after" and "Add a parallel step" in a step's menu, and a
// handle dragged from one step onto another.

/// Ship's steps in a line: Payments, then Search, then Launch. Launch is
/// someone else's to write unless [launchWritable].
class _FlowApi extends TreeApi {
  _FlowApi({super.canWritePlan, super.canWrite, this.launchWritable = true});

  final bool launchWritable;
  final created = <({String title, String? parentId})>[];

  RoadmapNode _step(String id, String title, int pos, List<String> on) =>
      RoadmapNode(
        id: id,
        projectId: 'p1',
        parentId: 'r',
        title: title,
        description: '',
        status: 'pending',
        position: pos,
        dependsOn: on,
        canWrite: id == 'c' ? launchWritable && canWrite : canWrite,
      );

  @override
  List<RoadmapNode> get all => [
    node('r', 'Ship', null),
    _step('a', 'Payments', 0, []),
    _step('b', 'Search', 1, ['a']),
    _step('c', 'Launch', 2, ['b']),
  ];

  @override
  Future<RoadmapNode> createRoadmapNode(
    String projectId,
    String title, {
    String? parentId,
  }) async {
    created.add((title: title, parentId: parentId));
    return _step('new', title, 3, []);
  }

  /// The new step is not in [all], which the base fake answers from.
  @override
  Future<RoadmapNode> addRoadmapDependency(String nodeId, String on) async {
    dependencyCalls.add((nodeId: nodeId, on: on, remove: false));
    return _step(nodeId, nodeId, 0, [on]);
  }
}

Future<void> _canvas(WidgetTester tester, _FlowApi api) async {
  final router = await pumpTreeRouter(tester, treeSession(api));
  router.go('/projects/p1/roadmap?view=canvas');
  await tester.pumpAndSettle();
}

/// Types [title] into the open form and confirms it.
Future<void> _name(WidgetTester tester, String title) async {
  await tester.enterText(
    find.descendant(of: find.byType(Dialog), matching: find.byType(TextField)),
    title,
  );
  await tester.tap(find.text('Add').last);
  await tester.pumpAndSettle();
}

/// Opens [id]'s step menu and picks [item].
Future<void> _menu(WidgetTester tester, String id, String item) async {
  await tester.tap(
    find.descendant(
      of: stepBox(id),
      matching: find.byType(PopupMenuButton<String>),
    ),
  );
  await tester.pumpAndSettle();
  await tester.tap(find.text(item).last);
  await tester.pumpAndSettle();
}

List<String> _links(_FlowApi api) => [
  for (final c in api.dependencyCalls) '${c.nodeId}->${c.on}',
];

void main() {
  testWidgets('the next-step slot adds a step after the last one', (
    tester,
  ) async {
    final api = _FlowApi();
    await _canvas(tester, api);
    await tester.tap(find.byKey(const Key('roadmap-next-step')));
    await tester.pumpAndSettle();
    expect(find.text('Next step'), findsWidgets);
    await _name(tester, 'Retro');

    expect(api.created.single, (title: 'Retro', parentId: 'r'));
    expect(_links(api), ['new->c']);
    expect(find.byType(SnackBar), findsNothing);
  });

  testWidgets('"Add a step after" waits on that step', (tester) async {
    final api = _FlowApi();
    await _canvas(tester, api);
    await _menu(tester, 'a', 'Add a step after');
    expect(find.text('A step after Payments'), findsOneWidget);
    await _name(tester, 'Payouts');

    expect(api.created.single.parentId, 'r');
    expect(_links(api), ['new->a']);
  });

  testWidgets('a parallel step shares the links; one it cannot write is '
      'named', (tester) async {
    final api = _FlowApi(launchWritable: false);
    await _canvas(tester, api);
    await _menu(tester, 'b', 'Add a parallel step');
    expect(find.text('A step beside Search'), findsOneWidget);
    await _name(tester, 'Filters');

    expect(_links(api), ['new->a'], reason: 'Launch is not the caller\'s');
    expect(find.text('Not linked, you cannot edit: Launch'), findsOneWidget);
  });

  testWidgets('dragging a handle onto a step makes it wait on this one', (
    tester,
  ) async {
    final api = _FlowApi();
    await _canvas(tester, api);
    final handle = find.byKey(const Key('roadmap-link-a'));
    final gesture = await tester.startGesture(tester.getCenter(handle));
    await gesture.moveBy(const Offset(40, 0));
    await tester.pump();
    expect(find.byKey(const Key('roadmap-link-line')), findsOneWidget);

    await gesture.moveTo(tester.getCenter(stepBox('c')));
    await gesture.up();
    await tester.pumpAndSettle();
    expect(_links(api), ['c->a']);
    expect(find.byKey(const Key('roadmap-link-line')), findsNothing);
  });

  testWidgets('a drop on empty canvas links nothing', (tester) async {
    final api = _FlowApi();
    await _canvas(tester, api);
    await tester.drag(
      find.byKey(const Key('roadmap-link-a')),
      const Offset(0, 400),
    );
    await tester.pumpAndSettle();
    expect(api.dependencyCalls, isEmpty);
  });

  testWidgets('a reader gets no slot and no add items', (tester) async {
    final api = _FlowApi(canWritePlan: false, canWrite: false);
    await _canvas(tester, api);
    expect(find.byKey(const Key('roadmap-next-step')), findsNothing);
    await tester.tap(
      find.descendant(
        of: stepBox('a'),
        matching: find.byType(PopupMenuButton<String>),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Add a step after'), findsNothing);
    expect(find.text('Add a parallel step'), findsNothing);
  });
}
