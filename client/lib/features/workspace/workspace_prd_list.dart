import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../chrome/loop_focus.dart';
import '../../chrome/time_ago.dart';
import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';
import '../../theme/cobalt_theme.dart';
import '../../theme/field_card.dart';

ShadeTone _prdTone(String status) => switch (status) {
  'approved' => ShadeTone.done,
  'reviewing' => ShadeTone.progress,
  'rejected' => ShadeTone.danger,
  _ => ShadeTone.pending,
};

/// Every PRD of the workspace with its status, author and linked project, so
/// a teammate or the admin who approves it can open one they did not write
/// (rv5-FE-02). The page reads [prds] with the board; null (a failed read)
/// shows nothing.
class WorkspacePrdList extends StatelessWidget {
  const WorkspacePrdList({
    super.key,
    required this.prds,
    required this.projects,
  });

  final List<WorkspacePrd>? prds;
  final List<Project> projects;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final titles = {for (final p in projects) p.id: p.title};
    final prds = this.prds;
    if (prds == null) return const SizedBox.shrink();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(l10n.t('workspace.prds').toUpperCase(), style: eyebrow(context)),
        const SizedBox(height: Cobalt.space3),
        if (prds.isEmpty)
          Text(
            l10n.t('workspace.prdsEmpty'),
            style: theme.textTheme.bodyMedium,
          ),
        for (final p in prds) ...[
          FieldCard(
            onTap: () {
              context.read<LoopFocus>().rememberPrd(p.id);
              context.go('/prd/${p.id}');
            },
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
                        style: theme.textTheme.titleMedium,
                      ),
                    ),
                    ShadeChip(
                      label: l10n.t('prd.status.${p.status}'),
                      tone: _prdTone(p.status),
                    ),
                  ],
                ),
                const SizedBox(height: Cobalt.space1),
                Text(
                  [
                    l10n.t('workspace.prdBy', {'name': p.authorName}),
                    timeAgo(p.updatedAt, l10n.t),
                  ].join(' · '),
                  style: theme.textTheme.bodySmall,
                ),
                if (p.projectId != null)
                  Text(
                    titles[p.projectId] == null
                        ? l10n.t('workspace.prdLinked')
                        : l10n.t('workspace.prdProject', {
                            'title': titles[p.projectId],
                          }),
                    style: theme.textTheme.bodySmall,
                  ),
              ],
            ),
          ),
          const SizedBox(height: Cobalt.space3),
        ],
      ],
    );
  }
}
