import 'package:flutter/material.dart';

import '../../theme/cobalt.dart';

/// CQ-01 (review 2): the Bands view and the Progress view render the same
/// band chrome — container, 72px uppercase label column, row spacing,
/// 240px cards, '—' empty state. One scaffold so a visual tweak to one
/// view can't silently skip the other; callers pass only their item cards.
class RoadmapBandScaffold extends StatelessWidget {
  const RoadmapBandScaffold({
    super.key,
    required this.label,
    required this.color,
    required this.itemCards,
    this.labelColor,
  });

  final String label;
  final Color color;
  final Color? labelColor;
  final Iterable<Widget> itemCards;

  static const cardWidth = 240.0;
  static const gap = 12.0;

  /// Rows of cards, each as tall as its tallest card, so a row's bottom
  /// edges line up instead of reading ragged (design QA 2026-11-21).
  static Widget _rows(List<Widget> items, double width) {
    final perRow = ((width + gap) / (cardWidth + gap)).floor().clamp(1, 99);
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
                  // it lays out at; a phone narrows it.
                  SizedBox(width: width.clamp(0, cardWidth), child: items[j]),
                ],
              ],
            ),
          ),
        ],
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final items = itemCards.toList();
    return Container(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 16),
      decoration: BoxDecoration(
        color: color,
        borderRadius: BorderRadius.circular(Cobalt.radiusHero),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 72,
            child: Text(
              label.toUpperCase(),
              style: theme.textTheme.labelSmall?.copyWith(
                color: labelColor,
                letterSpacing: 0.6,
              ),
            ),
          ),
          Expanded(
            child: items.isEmpty
                ? Text(
                    '—',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: labelColor,
                    ),
                  )
                : LayoutBuilder(
                    builder: (context, box) => _rows(items, box.maxWidth),
                  ),
          ),
        ],
      ),
    );
  }
}
