import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/client.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/chrome/loop_focus.dart';
import 'package:insideout/features/workspace/workspace_page.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/session/session.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_theme.dart';
import 'package:insideout/widgets/pressable_block.dart';
import 'package:provider/provider.dart';

// Block-craft follow-up: the workspace quick-capture control was a stock
// FloatingActionButton — flat (it cannot cast or press) and unnamed (no
// tooltip, no semantic label). It is now a PressableBlock primary block with
// a localized tooltip and screen-reader name. Driven through the real
// WorkspacePage so a control that is defined but never mounted still fails.
class _FakeApi extends ApiClient {
  _FakeApi()
    : super(
        baseUrl: 'http://127.0.0.1:9/api/v1',
        readAccess: () async => null,
        readRefresh: () async => null,
        writeTokens: (a, r) async {},
        clearTokens: () async {},
      );

  @override
  Future<Workspace> getWorkspace(String id) async => Workspace.fromJson({
    'id': 'ws1',
    'title': 'Capture WS',
    'description': '',
    'code': 'CODE12',
    'status': 'active',
    'memberCount': 1,
    'myRole': 'admin',
    'createdAt': '2026-01-01T00:00:00Z',
  });

  // An empty board keeps the page free of project cards (and their progress
  // fetches); the only "New project" tooltip left is the capture control.
  @override
  Future<List<Project>> listProjects(String workspaceId) async => <Project>[];
}

const _labels = {'en-US': 'New project', 'zh-CN': '新项目'};

Future<void> _pump(
  WidgetTester tester, {
  String locale = 'en-US',
  bool dark = false,
}) async {
  await tester.binding.setSurfaceSize(const Size(900, 700));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  final session = Session()
    ..ready = true
    ..api = _FakeApi();
  await tester.pumpWidget(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: session),
        ChangeNotifierProvider(
          create: (_) => Appearance(
            store: MemorySettingsStore(),
            locale: locale,
            dark: dark,
          ),
        ),
        ChangeNotifierProvider(create: (_) => LoopFocus()),
      ],
      child: MaterialApp(
        theme: cobaltTheme(
          dark ? Cobalt.dark : Cobalt.light,
          brightness: dark ? Brightness.dark : Brightness.light,
        ),
        home: const WorkspacePage(id: 'ws1'),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  for (final MapEntry(key: locale, value: label) in _labels.entries) {
    testWidgets('the capture control is a named button ($locale)', (
      tester,
    ) async {
      // Released inside the body: the framework verifies handles before
      // tearDowns run.
      final semantics = tester.ensureSemantics();
      try {
        await _pump(tester, locale: locale);

        // The tooltip is the localized action name, not a hardcoded string.
        final control = find.byTooltip(label);
        expect(control, findsOneWidget);

        // And a screen reader gets the same name on an actionable button.
        final data = tester
            .getSemantics(
              find.descendant(
                of: control,
                matching: find.byType(PressableBlock),
              ),
            )
            .getSemanticsData();
        expect(data.label, label);
        expect(data.flagsCollection.isButton, isTrue);
        expect(data.hasAction(SemanticsAction.tap), isTrue);
      } finally {
        semantics.dispose();
      }
    });
  }

  for (final dark in [false, true]) {
    final theme = dark ? 'dark' : 'light';
    final brightness = dark ? Brightness.dark : Brightness.light;
    final palette = dark ? Cobalt.dark : Cobalt.light;

    testWidgets('it casts at rest, steps to castSm on hover and presses flat '
        '($theme)', (tester) async {
      await _pump(tester, dark: dark);
      final control = find.byTooltip('New project');
      final block = find
          .descendant(of: control, matching: find.byType(AnimatedContainer))
          .first;
      BoxDecoration shadowed() =>
          tester.widget<AnimatedContainer>(block).decoration! as BoxDecoration;
      Offset travel() {
        final t = tester
            .widget<AnimatedContainer>(block)
            .transform!
            .getTranslation();
        return Offset(t.x, t.y);
      }

      // Rest: the hard (4,4) cast, no travel.
      expect(shadowed().boxShadow, cast(brightness));
      expect(travel(), Offset.zero);

      // Hover: the cast shrinks to (2,2) and the block travels into it.
      final mouse = await tester.createGesture(kind: PointerDeviceKind.mouse);
      await mouse.addPointer(location: Offset.zero);
      addTearDown(mouse.removePointer);
      await mouse.moveTo(tester.getCenter(control));
      await tester.pumpAndSettle();
      expect(shadowed().boxShadow, castSm(brightness));
      expect(travel(), Cobalt.hoverOffset);

      // Active: shadow gone, the block sits (4,4) into the page.
      await mouse.down(tester.getCenter(control));
      await tester.pumpAndSettle();
      expect(shadowed().boxShadow, isEmpty);
      expect(travel(), Cobalt.activeOffset);
      await mouse.up();
      await tester.pumpAndSettle();
    });

    testWidgets('the face is a 56px square primary block edged in ink '
        '($theme)', (tester) async {
      await _pump(tester, dark: dark);
      final control = find.byTooltip('New project');

      // Same footprint as the FAB it replaces, so the board's reserved bottom
      // inset (derived from the shared size) still clears it.
      expect(tester.getSize(control), const Size.square(56));

      final face = tester
          .widgetList<DecoratedBox>(
            find.descendant(of: control, matching: find.byType(DecoratedBox)),
          )
          .map((d) => d.decoration)
          .whereType<BoxDecoration>()
          .firstWhere((d) => d.border != null);
      expect(face.color, palette.btn);
      expect(
        face.border,
        Border.all(color: Cobalt.lightInk, width: Cobalt.borderContainer),
      );
      expect(face.borderRadius, isNull);
      final icon = tester.widget<Icon>(
        find.descendant(of: control, matching: find.byIcon(Icons.add)),
      );
      expect(icon.color, palette.btnFg);
    });
  }

  testWidgets('tapping it opens the New project form', (tester) async {
    await _pump(tester);

    await tester.tap(find.byTooltip('New project'));
    await tester.pumpAndSettle();

    expect(find.byType(TextField), findsOneWidget);
    expect(find.text('Create'), findsOneWidget);
  });
}
