import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';
import '../../theme/field_card.dart';

/// One root of "My work": its title and path, its Now items with their
/// pressure, and what it waits on and blocks. A tap opens its Canvas.
class MyWorkRootCard extends StatelessWidget {
  const MyWorkRootCard(this.root, {super.key});

  final MyRoadmapRoot root;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final e = root.read;
    final small = theme.textTheme.bodySmall;
    // The project root opens the project roadmap; any other its own tree.
    final at = e.node.parentId == null ? '' : '/${e.node.id}';
    return FieldCard(
      padding: const EdgeInsets.all(Cobalt.space4),
      onTap: () =>
          context.go('/projects/${e.projectId}/roadmap$at?view=canvas'),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(e.node.title, style: theme.textTheme.titleMedium),
              ),
              ShadeChip(
                label: l10n.t('roadmap.status.${e.node.status}'),
                tone: ShadeChip.fromStatus(e.node.status),
              ),
            ],
          ),
          const SizedBox(height: Cobalt.space2),
          Text(
            [e.projectName, ...e.ancestors.map((a) => a.title)].join(' › '),
            style: small,
          ),
          // A delegated leaf counts itself, so a roll-up says nothing about
          // it. shortcut: the mine read carries no child count, so a branch
          // over one leaf hides it too; add one if needed.
          if (e.node.rollup.leaves > 1)
            Text(
              l10n.t('roadmap.rollup', {
                'done': e.node.rollup.done,
                'leaves': e.node.rollup.leaves,
                'validated': e.node.rollup.validated,
              }),
              style: small,
            ),
          for (final item in root.now) _NowRow(item),
          if (root.waitsOn.isNotEmpty)
            _LinkLine(
              l10n.t('roadmap.waitsOn', {'titles': _names(root.waitsOn)}),
            ),
          if (root.blocking.isNotEmpty)
            _LinkLine(
              l10n.t('home.myWork.blocking', {'titles': _names(root.blocking)}),
            ),
        ],
      ),
    );
  }
}

String _names(List<RoadmapLink> links) => [
  for (final l in links)
    l.ownerName.isEmpty ? l.title : '${l.title} (${l.ownerName})',
].join(', ');

class _NowRow extends StatelessWidget {
  const _NowRow(this.item);

  final ProgressItem item;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final pressure = item.pressure;
    return Padding(
      padding: const EdgeInsets.only(top: Cobalt.space2),
      child: Row(
        children: [
          Expanded(
            child: Text(
              item.title,
              style: Theme.of(context).textTheme.bodyMedium,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
          // "normal" is the quiet default; Progress hides it too.
          if (pressure != null && pressure.isNotEmpty && pressure != 'normal')
            ShadeChip(
              label: l10n.t('roadmap.pressure.$pressure'),
              tone: pressure == 'overdue' || pressure == 'high_risk'
                  ? ShadeTone.danger
                  : ShadeTone.neutral,
            ),
        ],
      ),
    );
  }
}

class _LinkLine extends StatelessWidget {
  const _LinkLine(this.text);

  final String text;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(top: Cobalt.space1),
    child: Text(
      text,
      style: Theme.of(context).textTheme.bodySmall,
      maxLines: 2,
      overflow: TextOverflow.ellipsis,
    ),
  );
}
