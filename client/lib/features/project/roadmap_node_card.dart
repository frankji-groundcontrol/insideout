import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../chrome/time_ago.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/block_field.dart';
import '../../theme/block_form.dart';
import '../../theme/block_modal.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';
import '../../theme/field_card.dart';
import '../../widgets/guarded_action.dart';

part 'roadmap_expand_draft.dart';
part 'roadmap_node_menu.dart';

class RoadmapNodeCard extends StatelessWidget {
  const RoadmapNodeCard({
    super.key,
    required this.projectId,
    required this.node,
    required this.onReload,
    this.compact = false,
    this.childProgress,
    this.descendants = 0,
  });

  final String projectId;
  final RoadmapNode node;
  final Future<void> Function() onReload;
  final bool compact;

  /// Nodes beneath this one; the server deletes the branch with it.
  final int descendants;

  /// Share of this node's children that are done (0..1), or null for a
  /// leaf — leaves show no bar rather than an invented percentage. Only a
  /// caller holding the whole tree can compute it.
  final double? childProgress;

  /// done-children share of [n] within [all], or null when [n] is a leaf.
  static double? childDoneShare(List<RoadmapNode> all, RoadmapNode n) {
    final children = all.where((o) => o.parentId == n.id).toList();
    if (children.isEmpty) return null;
    final done = children.where((c) => c.status == 'done').length;
    return done / children.length;
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    return FieldCard(
      padding: const EdgeInsets.fromLTRB(16, 14, 8, 14),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              ShadeChip(
                label: l10n.t('roadmap.status.${node.status}'),
                tone: ShadeChip.fromStatus(node.status),
              ),
              const Spacer(),
              _NodeMenu(
                projectId: projectId,
                node: node,
                onReload: onReload,
                descendants: descendants,
              ),
            ],
          ),
          const SizedBox(height: 10),
          Text(node.title, style: theme.textTheme.titleMedium),
          if (node.editorName != null)
            Text(node.editorName!, style: theme.textTheme.bodySmall),
          if (node.deadline != null)
            Padding(
              padding: const EdgeInsets.only(top: 6),
              child: ShadeChip(
                label:
                    '${isoDate(node.deadline!)}${node.pressure != null && node.pressure != 'normal' ? ' · ${l10n.t('roadmap.pressure.${node.pressure}')}' : ''}',
                tone: node.pressure == 'overdue' || node.pressure == 'high_risk'
                    ? ShadeTone.danger
                    : ShadeTone.neutral,
              ),
            ),
          if (node.acceptance.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 6),
              child: Text(
                '${l10n.t('roadmap.acceptance')}: ${node.acceptance}',
                style: theme.textTheme.bodySmall,
              ),
            ),
          if (node.ownerName.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 4),
              child: Text(
                '${l10n.t('roadmap.owner')}: ${node.ownerName}',
                style: theme.textTheme.bodySmall,
              ),
            ),
          // The milestone stays readable while it is the live status; the
          // row keeps the reason after a reopen too (it is history), but
          // the card speaks only for the current state.
          if (node.status == 'validated' && node.validationReason.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 4),
              child: Text(
                '${l10n.t('roadmap.validatedBecause')}: ${node.validationReason}',
                style: theme.textTheme.bodySmall,
              ),
            ),
          if (childProgress != null) ...[
            const SizedBox(height: 12),
            ClipRRect(
              borderRadius: BorderRadius.circular(Cobalt.radiusPill),
              child: LinearProgressIndicator(
                value: childProgress,
                minHeight: 4,
              ),
            ),
          ],
        ],
      ),
    );
  }
}
