import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../chrome/time_ago.dart';
import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import 'roadmap_band_scaffold.dart';

/// FEAT-05 — PRODUCT.md's default Progress view, rendered from
/// GET /roadmap/progress (server truth): Now holds only deadlined
/// in-progress work with its pressure state, at most three justified
/// Next, needsDeadline holds deadline-less in-progress work, and Done is
/// counted. The Bands/Canvas views stay for structure planning.
class RoadmapProgressView extends StatelessWidget {
  const RoadmapProgressView({super.key, required this.progress, required this.emptyLabel});

  final RoadmapProgress progress;
  final String emptyLabel;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    final empty = progress.now.isEmpty && progress.next.isEmpty && progress.needsDeadline.isEmpty && progress.doneCount == 0;
    return ListView(
      padding: const EdgeInsets.fromLTRB(24, 8, 24, 140),
      children: [
        if (empty)
          Center(child: Text(emptyLabel, style: theme.textTheme.bodyLarge))
        else ...[
          Align(
            alignment: Alignment.centerLeft,
            child: Chip(
              avatar: Icon(Icons.check_circle_outline, size: 18, color: palette.brand),
              label: Text(l10n.t('roadmap.doneCount', {'n': progress.doneCount})),
            ),
          ),
          const SizedBox(height: 4),
          _ProgressBand(label: l10n.t('roadmap.bandNow'), color: palette.surfaceSunken, items: progress.now),
          const SizedBox(height: 16),
          _ProgressBand(label: l10n.t('roadmap.needsDeadline'), color: palette.brandSubtle, items: progress.needsDeadline),
          const SizedBox(height: 16),
          _ProgressBand(label: l10n.t('roadmap.bandNext'), color: palette.brand, labelColor: palette.btnFg, items: progress.next),
        ],
      ],
    );
  }
}

class _ProgressBand extends StatelessWidget {
  const _ProgressBand({
    required this.label,
    required this.color,
    required this.items,
    this.labelColor,
  });

  final String label;
  final Color color;
  final Color? labelColor;
  final List<ProgressItem> items;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    return RoadmapBandScaffold(
      label: label,
      color: color,
      labelColor: labelColor,
      itemCards: [
        for (final item in items)
          SizedBox(
            width: 240,
            child: Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: palette.surfaceRaised,
                borderRadius: BorderRadius.circular(Cobalt.radiusCard),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(item.title, style: theme.textTheme.titleSmall, maxLines: 2, overflow: TextOverflow.ellipsis),
                  const SizedBox(height: 6),
                  Row(
                    children: [
                      if (item.deadline != null)
                        Padding(
                          padding: const EdgeInsets.only(right: 8),
                          child: Text(isoDate(item.deadline!), style: theme.textTheme.bodySmall),
                        ),
                      if (item.pressure != null && item.pressure!.isNotEmpty)
                        Text(
                          l10n.t('roadmap.pressure.${item.pressure}'),
                          style: theme.textTheme.labelSmall?.copyWith(
                            color: item.pressure == 'overdue' || item.pressure == 'high_risk' ? palette.fgDanger : palette.brand,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                    ],
                  ),
                ],
              ),
            ),
          ),
      ],
    );
  }
}
