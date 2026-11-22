import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/errors.dart';
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
import 'roadmap_completion_prompt.dart';
import 'roadmap_dependency_picker.dart';
import 'roadmap_node_detail.dart';

part 'roadmap_expand_draft.dart';
part 'roadmap_node_delegate.dart';
part 'roadmap_node_edit.dart';
part 'roadmap_node_menu.dart';

class RoadmapNodeCard extends StatelessWidget {
  const RoadmapNodeCard({
    super.key,
    required this.projectId,
    required this.node,
    required this.onReload,
    this.compact = false,
    this.descendants = 0,
    this.planWriter = false,
    this.parentWritable = false,
    this.inheritedOwner,
    this.openedRoot = false,
  });

  final String projectId;
  final RoadmapNode node;
  final Future<void> Function() onReload;
  final bool compact;

  /// Nodes beneath this one; the server deletes the branch with it.
  final int descendants;

  /// The caller writes the whole plan (project owner or workspace admin),
  /// and the caller may write this node's parent (owns a branch above it):
  /// together they decide the move, delete and delegate items.
  final bool planWriter;
  final bool parentWritable;

  /// The owner this node holds through an owned ancestor, when it has none
  /// of its own (roadmapInheritedOwner).
  final ({String name, String via})? inheritedOwner;

  /// This node is the root of the opened tree: deleting it leaves for the
  /// parent's tree, since this route no longer exists.
  final bool openedRoot;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    // The server's roll-up over every leaf below (validated counts as done);
    // a leaf shows no bar rather than an invented percentage.
    final rollup = node.rollup;
    final showRollup = descendants > 0 && rollup.leaves > 0;
    final ownerLabel = roadmapOwnerLabel(l10n.t, node, inheritedOwner);
    return FieldCard(
      // A branch opens as its own tree; a leaf has nothing below it, so it
      // opens its read-only detail instead.
      onTap: descendants > 0
          ? () => GoRouter.maybeOf(
              context,
            )?.go('/projects/$projectId/roadmap/${node.id}')
          : () => showRoadmapNodeDetail(
              context,
              node,
              inheritedOwner: inheritedOwner,
            ),
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
                planWriter: planWriter,
                parentWritable: parentWritable,
                inheritedOwner: inheritedOwner,
                openedRoot: openedRoot,
              ),
            ],
          ),
          const SizedBox(height: 10),
          Text(node.title, style: theme.textTheme.titleMedium),
          // The last editor, unlabelled; a compact band card leaves it out so
          // it does not read as a second owner line.
          if (node.editorName != null && !compact)
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
          if (ownerLabel != null)
            Padding(
              padding: const EdgeInsets.only(top: 4),
              child: Text(
                '${l10n.t('roadmap.owner')}: $ownerLabel',
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
          if (showRollup) ...[
            const SizedBox(height: 12),
            ClipRRect(
              borderRadius: BorderRadius.circular(Cobalt.radiusPill),
              child: LinearProgressIndicator(
                value: rollup.done / rollup.leaves,
                minHeight: 4,
                semanticsLabel: l10n.t('roadmap.rollupShort', {
                  'done': rollup.done,
                  'leaves': rollup.leaves,
                }),
              ),
            ),
            const SizedBox(height: 4),
            Text(
              l10n.t('roadmap.rollup', {
                'done': rollup.done,
                'leaves': rollup.leaves,
                'validated': rollup.validated,
              }),
              style: theme.textTheme.bodySmall,
            ),
          ],
        ],
      ),
    );
  }
}
