import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../chrome/time_ago.dart';
import '../../session/appearance.dart';
import '../../theme/block_chrome.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';
import '../../theme/field_card.dart';

ShadeTone projectStatusTone(String status) => switch (status) {
  'done' => ShadeTone.done,
  'active' => ShadeTone.progress,
  'paused' || 'archived' => ShadeTone.locked,
  _ => ShadeTone.pending,
};

/// One project on the board: pressable card, title + quiet flag, status chip
/// with the live Now/Done count, and the latest-update meta line.
class BoardProjectCard extends StatelessWidget {
  const BoardProjectCard({
    super.key,
    required this.project,
    required this.progress,
    required this.onOpen,
  });

  final Project project;
  final Future<RoadmapProgress> progress;
  final VoidCallback onOpen;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final p = project;
    return FieldCard(
      onTap: onOpen,
      padding: const EdgeInsets.all(Cobalt.space4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  p.title,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: theme.textTheme.titleLarge,
                ),
              ),
              if (p.flagged)
                ShadeChip(
                  label: l10n.t('project.quiet'),
                  tone: ShadeTone.danger,
                ),
            ],
          ),
          const SizedBox(height: Cobalt.space2),
          Row(
            children: [
              ShadeChip(
                label: l10n.t('project.status.${p.status}'),
                tone: projectStatusTone(p.status),
              ),
              FutureBuilder<RoadmapProgress>(
                future: progress,
                builder: (context, psnap) {
                  if (!psnap.hasData) return const SizedBox.shrink();
                  return Padding(
                    padding: const EdgeInsets.only(left: Cobalt.space2),
                    child: Text(
                      l10n.t('workspace.nowDone', {
                        'now': psnap.data!.now.length,
                        'done': psnap.data!.doneCount,
                      }),
                      style: theme.textTheme.bodySmall,
                    ),
                  );
                },
              ),
            ],
          ),
          if (p.latestUpdateKind != null)
            Padding(
              padding: const EdgeInsets.only(top: Cobalt.space3),
              child: Row(
                children: [
                  BlockTag(
                    label: l10n.t('project.updateKind.${p.latestUpdateKind}'),
                  ),
                  const SizedBox(width: Cobalt.space2),
                  Expanded(
                    child: Text(
                      p.latestUpdateContent ?? '',
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: theme.textTheme.bodySmall,
                    ),
                  ),
                  if (p.latestUpdateAt != null)
                    Padding(
                      padding: const EdgeInsets.only(left: Cobalt.space2),
                      child: Text(
                        timeAgo(p.latestUpdateAt!, l10n.t),
                        style: theme.textTheme.bodySmall,
                      ),
                    ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}
