import 'package:flutter/material.dart';

import '../../theme/cobalt.dart';

/// One run of cards in a band, under an optional sub-heading (Bands' Later
/// groups its cards by top-level branch).
class RoadmapBandGroup {
  const RoadmapBandGroup(this.heading, this.cards);

  final String? heading;
  final List<Widget> cards;
}

/// CQ-01 (review 2): the Bands view and the Progress view render the same
/// band chrome — neutral lane, uppercase label, row spacing, cards that fill
/// each row, '—' empty state. One scaffold so a visual tweak to one view
/// can't silently skip the other; callers pass only their item cards.
class RoadmapBandScaffold extends StatelessWidget {
  RoadmapBandScaffold({
    super.key,
    required this.label,
    required Iterable<Widget> itemCards,
    this.now = false,
  }) : groups = [RoadmapBandGroup(null, itemCards.toList())];

  const RoadmapBandScaffold.grouped({
    super.key,
    required this.label,
    required this.groups,
    this.now = false,
  });

  final String label;
  final List<RoadmapBandGroup> groups;

  /// The Now band: DESIGN.md gives cobalt-500 to Now alone, so it gets a
  /// brand strip on its left edge and a cobalt label block.
  final bool now;

  static const minCardWidth = 220.0;
  static const gap = 12.0;
  static const nowStrip = 4.0;

  /// Centres [child] in the 1120 content column on wide screens.
  static Widget capped(Widget child) => Align(
    alignment: Alignment.topCenter,
    child: ConstrainedBox(
      constraints: const BoxConstraints(maxWidth: Cobalt.contentMaxWidth),
      child: SizedBox(width: double.infinity, child: child),
    ),
  );

  /// Rows of cards that fill the width, each row as tall as its tallest
  /// card, so a row's bottom edges line up instead of reading ragged
  /// (design QA 2026-11-21).
  static Widget _rows(List<Widget> items, double width) {
    final perRow = ((width + gap) / (minCardWidth + gap)).floor().clamp(1, 99);
    final cardWidth = (width - gap * (perRow - 1)) / perRow;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (var i = 0; i < items.length; i += perRow) ...[
          if (i > 0) const SizedBox(height: gap),
          IntrinsicHeight(
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                for (var j = i; j < i + perRow && j < items.length; j++) ...[
                  if (j > i) const SizedBox(width: gap),
                  // A tight width, so the row measures each card at the width
                  // it lays out at.
                  SizedBox(width: cardWidth, child: items[j]),
                ],
              ],
            ),
          ),
        ],
      ],
    );
  }

  Widget _content(ThemeData theme, Cobalt palette) {
    if (groups.every((g) => g.cards.isEmpty)) {
      return Text('—', style: theme.textTheme.bodySmall);
    }
    final shown = groups.where((g) => g.cards.isNotEmpty).toList();
    return LayoutBuilder(
      builder: (context, box) => Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          for (final (i, g) in shown.indexed) ...[
            if (i > 0) const SizedBox(height: Cobalt.space4),
            if (g.heading != null)
              Padding(
                padding: const EdgeInsets.only(bottom: Cobalt.space2),
                child: Text(
                  g.heading!,
                  key: ValueKey('roadmap-band-group-${g.heading}'),
                  style: theme.textTheme.labelMedium?.copyWith(
                    color: palette.fgSecondary,
                  ),
                ),
              ),
            _rows(g.cards, box.maxWidth),
          ],
        ],
      ),
    );
  }

  Widget _label(ThemeData theme, Cobalt palette) => Container(
    key: const Key('roadmap-band-label'),
    padding: now
        ? const EdgeInsets.symmetric(horizontal: 6, vertical: 2)
        : EdgeInsets.zero,
    decoration: now ? BoxDecoration(color: palette.brand) : null,
    child: Text(
      label.toUpperCase(),
      style: theme.textTheme.labelSmall?.copyWith(
        color: now ? palette.btnFg : null,
        letterSpacing: 0.6,
      ),
    ),
  );

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    final label = _label(theme, palette);
    final content = _content(theme, palette);
    // On a phone a 72px side column took a quarter of the lane.
    final stacked = MediaQuery.sizeOf(context).width < Cobalt.bpNarrow;
    final slab = Container(
      key: const Key('roadmap-band-slab'),
      padding: const EdgeInsets.all(Cobalt.space4),
      decoration: BoxDecoration(
        color: palette.surfaceSunken,
        borderRadius: BorderRadius.circular(Cobalt.radiusHero),
      ),
      child: stacked
          ? Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Align(alignment: Alignment.centerLeft, child: label),
                const SizedBox(height: Cobalt.space3),
                content,
              ],
            )
          : Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                SizedBox(
                  width: 72,
                  child: Align(alignment: Alignment.topLeft, child: label),
                ),
                Expanded(child: content),
              ],
            ),
    );
    return capped(
      now
          ? Stack(
              fit: StackFit.passthrough,
              children: [
                slab,
                Positioned(
                  left: 0,
                  top: 0,
                  bottom: 0,
                  child: Container(
                    key: const Key('roadmap-band-now-strip'),
                    width: nowStrip,
                    color: palette.brand,
                  ),
                ),
              ],
            )
          : slab,
    );
  }
}
