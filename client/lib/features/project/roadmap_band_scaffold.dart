import 'package:flutter/material.dart';

import '../../theme/cobalt.dart';

/// CQ-01 (review 2): the Bands view and the Progress view render the same
/// band chrome — container, 72px uppercase label column, Wrap spacing,
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
                : Wrap(spacing: 12, runSpacing: 12, children: items),
          ),
        ],
      ),
    );
  }
}
