import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../chrome/time_ago.dart';
import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';
import '../../theme/field_card.dart';
import 'roadmap_graph_layout.dart';
import 'roadmap_node_detail.dart';

/// One node on the Canvas graph: a fixed-size block with its status, title,
/// owner, roll-up, deadline and pressure, the "+N" of a subtree too deep to
/// draw, and what it waits on out of view. Pressing it opens the node as its
/// own tree, which is what its owner sees as the root.
class RoadmapGraphNode extends StatelessWidget {
  const RoadmapGraphNode({
    super.key,
    required this.projectId,
    required this.node,
    this.branch = false,
    this.hidden = 0,
    this.waitsOn = const [],
    this.inheritedOwner,
    this.menu,
  });

  final String projectId;
  final RoadmapNode node;

  /// The node has children: its roll-up counts their leaves.
  final bool branch;

  /// Nodes below this one that the Canvas does not draw.
  final int hidden;

  /// Titles of the nodes this one waits on that are not drawn.
  final List<String> waitsOn;
  final ({String name, String via})? inheritedOwner;

  /// The node menu, for the edit, link and delete actions.
  final Widget? menu;

  @override
  Widget build(BuildContext context) {
    final t = context.watch<Appearance>().t;
    final theme = Theme.of(context);
    final small = theme.textTheme.bodySmall;
    final owner = roadmapOwnerLabel(t, node, inheritedOwner);
    final rollup = node.rollup;
    final pressure = node.pressure == null || node.pressure == 'normal'
        ? ''
        : ' · ${t('roadmap.pressure.${node.pressure}')}';
    Text line(String text) =>
        Text(text, style: small, maxLines: 1, overflow: TextOverflow.ellipsis);
    final box = FieldCard(
      onTap: () => GoRouter.maybeOf(
        context,
      )?.go('/projects/$projectId/roadmap/${node.id}?view=canvas'),
      padding: const EdgeInsets.all(Cobalt.space3),
      // The box is fixed; at a large text scale its foot clips, not overflows.
      child: OverflowBox(
        alignment: Alignment.topLeft,
        maxHeight: double.infinity,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          spacing: Cobalt.space1,
          children: [
            SizedBox(
              height: Cobalt.controlHeightDense,
              child: Row(
                children: [
                  ShadeChip(
                    label: t('roadmap.status.${node.status}'),
                    tone: ShadeChip.fromStatus(node.status),
                  ),
                  const Spacer(),
                  ?menu,
                ],
              ),
            ),
            Text(
              node.title,
              style: theme.textTheme.titleSmall,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
            ),
            if (owner != null) line('${t('roadmap.owner')}: $owner'),
            if (branch && rollup.leaves > 0)
              line(
                t('roadmap.rollup', {
                  'done': rollup.done,
                  'leaves': rollup.leaves,
                  'validated': rollup.validated,
                }),
              ),
            if (waitsOn.isNotEmpty)
              line(t('roadmap.waitsOn', {'titles': waitsOn.join(', ')})),
            if (node.deadline != null)
              ShadeChip(
                label: '${isoDate(node.deadline!)}$pressure',
                tone: node.pressure == 'overdue' || node.pressure == 'high_risk'
                    ? ShadeTone.danger
                    : ShadeTone.neutral,
              ),
          ],
        ),
      ),
    );
    final sized = SizedBox(height: RoadmapGraphLayout.boxHeight, child: box);
    if (hidden == 0) return sized;
    // The deeper subtree hangs below the box: a stub of connector and its
    // count, where the next level would start.
    return Column(
      children: [
        sized,
        Container(
          width: Cobalt.borderContainer,
          height: Cobalt.space4,
          color: Cobalt.of(context).strokeStrong,
        ),
        ShadeChip(
          label: t('roadmap.graphBelow', {'n': hidden}),
          tone: ShadeTone.neutral,
        ),
      ],
    );
  }
}
