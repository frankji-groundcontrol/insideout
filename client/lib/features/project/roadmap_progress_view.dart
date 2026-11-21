import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../chrome/product_dock.dart';
import '../../chrome/time_ago.dart';
import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';
import '../../theme/field_card.dart';
import 'roadmap_band_scaffold.dart';

/// FEAT-05 — PRODUCT.md's default Progress view, rendered from
/// GET /roadmap/progress (server truth): Now holds only deadlined
/// in-progress work with its pressure state, at most three justified
/// Next, needsDeadline holds deadline-less in-progress work, and Done is
/// counted. The Bands/Canvas views stay for structure planning.
class RoadmapProgressView extends StatelessWidget {
  const RoadmapProgressView({
    super.key,
    required this.progress,
    required this.emptyLabel,
  });

  final RoadmapProgress progress;
  final String emptyLabel;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    final empty =
        progress.now.isEmpty &&
        progress.next.isEmpty &&
        progress.needsDeadline.isEmpty &&
        progress.doneCount == 0;
    return ListView(
      padding: const EdgeInsets.fromLTRB(24, 8, 24, ProductDock.clearance),
      children: [
        if (empty)
          Center(child: Text(emptyLabel, style: theme.textTheme.bodyLarge))
        else ...[
          RoadmapBandScaffold.capped(
            Align(
              alignment: Alignment.centerLeft,
              child: Container(
                padding: const EdgeInsets.symmetric(
                  horizontal: Cobalt.space3,
                  vertical: Cobalt.space2,
                ),
                decoration: BoxDecoration(
                  color: palette.surfaceRaised,
                  border: Border.all(
                    color: palette.strokeSubtle,
                    width: Cobalt.borderContainer,
                  ),
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(
                      Icons.check_circle_outline,
                      size: 16,
                      color: palette.brand,
                    ),
                    const SizedBox(width: Cobalt.space2),
                    Text(
                      l10n.t('roadmap.doneCount', {'n': progress.doneCount}),
                      style: theme.textTheme.labelMedium,
                    ),
                  ],
                ),
              ),
            ),
          ),
          const SizedBox(height: 4),
          _ProgressBand(
            label: l10n.t('roadmap.bandNow'),
            now: true,
            items: progress.now,
          ),
          if (progress.needsDeadline.isNotEmpty) ...[
            const SizedBox(height: 16),
            _ProgressBand(
              label: l10n.t('roadmap.needsDeadline'),
              items: progress.needsDeadline,
            ),
          ],
          const SizedBox(height: 16),
          _ProgressBand(
            label: l10n.t('roadmap.bandNext'),
            items: progress.next,
          ),
          if (progress.doneItems.isNotEmpty) ...[
            const SizedBox(height: 16),
            _ProgressBand(
              label: l10n.t('roadmap.bandDone'),
              items: progress.doneItems,
            ),
          ],
        ],
      ],
    );
  }
}

class _ProgressBand extends StatelessWidget {
  const _ProgressBand({
    required this.label,
    required this.items,
    this.now = false,
  });

  final String label;
  final bool now;
  final List<ProgressItem> items;

  @override
  Widget build(BuildContext context) {
    return RoadmapBandScaffold(
      label: label,
      now: now,
      itemCards: [for (final item in items) _ProgressCard(item)],
    );
  }
}

class _ProgressCard extends StatelessWidget {
  const _ProgressCard(this.item);

  final ProgressItem item;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final pressure = item.pressure;
    return FieldCard(
      padding: const EdgeInsets.all(12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            item.title,
            style: theme.textTheme.titleSmall,
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
          ),
          if (item.ownerName.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 4),
              child: Text(
                '${l10n.t('roadmap.owner')}: ${item.ownerName}',
                style: theme.textTheme.bodySmall,
              ),
            ),
          if (item.acceptance.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 2),
              child: Text(
                '${l10n.t('roadmap.acceptance')}: ${item.acceptance}',
                style: theme.textTheme.bodySmall,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
            ),
          const SizedBox(height: 6),
          Wrap(
            spacing: Cobalt.space2,
            runSpacing: Cobalt.space1,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              if (item.deadline != null)
                Text(isoDate(item.deadline!), style: theme.textTheme.bodySmall),
              // "normal" is the quiet default; the Bands cards hide it too.
              if (pressure != null &&
                  pressure.isNotEmpty &&
                  pressure != 'normal')
                ShadeChip(
                  label: l10n.t('roadmap.pressure.$pressure'),
                  tone: pressure == 'overdue' || pressure == 'high_risk'
                      ? ShadeTone.danger
                      : ShadeTone.neutral,
                ),
            ],
          ),
        ],
      ),
    );
  }
}
