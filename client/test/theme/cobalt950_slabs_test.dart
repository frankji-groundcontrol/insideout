import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/models.dart';
import 'package:insideout/features/prd/versions_stack.dart';
import 'package:insideout/features/workspace/idea_poster.dart';
import 'package:insideout/session/appearance.dart';
import 'package:insideout/theme/cobalt.dart';
import 'package:insideout/theme/cobalt_theme.dart';
import 'package:insideout/theme/field_card.dart';
import 'package:insideout/widgets/pressable_block.dart';
import 'package:provider/provider.dart';

// DESIGN.md reserves cobalt-950 (brandSubtle in dark) for status-chip
// backgrounds; whole surfaces take surfaceRaised / surfaceSunken. The idea
// poster's gradient end and the versions stack's peek cards painted it as a
// slab. Rendered, not asserted on the palette: cobalt_test.dart still owns the
// token values, this owns where two surfaces take them from.
Future<void> _pump(
  WidgetTester tester,
  Widget child, {
  required bool dark,
}) async {
  await tester.binding.setSurfaceSize(const Size(900, 1000));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    ChangeNotifierProvider(
      create: (_) =>
          Appearance(store: MemorySettingsStore(), locale: 'en-US', dark: dark),
      child: MaterialApp(
        theme: cobaltTheme(
          dark ? Cobalt.dark : Cobalt.light,
          brightness: dark ? Brightness.dark : Brightness.light,
        ),
        home: Scaffold(body: child),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

double _contrast(Color a, Color b) {
  final la = a.computeLuminance();
  final lb = b.computeLuminance();
  final hi = la > lb ? la : lb;
  final lo = la > lb ? lb : la;
  return (hi + 0.05) / (lo + 0.05);
}

final _idea = Idea(
  id: 'i1',
  workspaceId: 'ws1',
  title: 'Night market',
  content: 'A map of the stalls that stay open late.',
  status: 'new',
  createdAt: '2026-10-01T00:00:00Z',
);

Widget _poster() => IdeaPoster(
  idea: _idea,
  onEdit: () {},
  onDrop: () {},
  onConvert: () {},
  onOpenPrd: () {},
);

PrdCommit _commit(int revision) => PrdCommit(
  id: 'c$revision',
  revision: revision,
  name: 'Commit $revision',
  primaryAudience: 'all',
  createdAt: '2026-10-0${revision}T00:00:00Z',
);

Widget _stack() {
  final commits = [_commit(3), _commit(2), _commit(1)];
  return VersionsStack(
    prdId: 'prd-1',
    commits: commits,
    selected: commits.first,
    readiness: null,
    audience: 'all',
    onSelect: (_) {},
    onAudience: (_) {},
    onCommit: () {},
  );
}

void main() {
  for (final dark in [false, true]) {
    final theme = dark ? 'dark' : 'light';
    final palette = dark ? Cobalt.dark : Cobalt.light;

    testWidgets('the idea poster fades raised to sunken, not to cobalt-950 '
        '($theme)', (tester) async {
      await _pump(tester, _poster(), dark: dark);

      final gradient = tester
          .widgetList<Container>(
            find.descendant(
              of: find.byType(IdeaPoster),
              matching: find.byType(Container),
            ),
          )
          .map((c) => c.decoration)
          .whereType<BoxDecoration>()
          .map((d) => d.gradient)
          .whereType<LinearGradient>()
          .single;
      expect(gradient.colors, [palette.surfaceRaised, palette.surfaceSunken]);
      if (dark) {
        expect(gradient.colors, isNot(contains(Cobalt.hue950)));
      }

      // The ink on the poster reads against both stops of the gradient.
      final title = tester
          .widget<Text>(find.text('NIGHT MARKET'))
          .style!
          .color!;
      final body = tester
          .widget<Text>(find.text('A map of the stalls that stay open late.'))
          .style!
          .color!;
      for (final stop in gradient.colors) {
        expect(_contrast(title, stop), greaterThanOrEqualTo(4.5));
        expect(_contrast(body, stop), greaterThanOrEqualTo(4.5));
      }
    });

    testWidgets('the versions peek cards are sunken, not cobalt-950 ($theme)', (
      tester,
    ) async {
      await _pump(tester, _stack(), dark: dark);

      // Only the commits behind the front card are pressable; their bodies
      // are the FieldCards inside those PressableBlocks.
      final peeks = tester.widgetList<FieldCard>(
        find.descendant(
          of: find.byType(PressableBlock),
          matching: find.byType(FieldCard),
        ),
      );
      expect(peeks, hasLength(2));
      for (final peek in peeks) {
        expect(peek.color, palette.surfaceSunken);
        if (dark) expect(peek.color, isNot(Cobalt.hue950));
      }
    });
  }
}
