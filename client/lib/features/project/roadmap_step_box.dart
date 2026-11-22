import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../chrome/time_ago.dart';
import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';
import '../../theme/field_card.dart';
import 'roadmap_node_detail.dart';

/// One step on the Canvas: a block as tall as its content, with the status,
/// title, owner, roll-up, how many items it holds, the deadline and pressure,
/// and what it waits on out of view. A step with items opens as its own
/// steps on the Canvas, the hierarchy one click down; a leaf opens its detail.
class RoadmapStepBox extends StatelessWidget {
  const RoadmapStepBox({
    super.key,
    required this.projectId,
    required this.node,
    this.inside = 0,
    this.waitsOn = const [],
    this.inheritedOwner,
    this.menu,
  });

  static const width = 248.0;

  final String projectId;
  final RoadmapNode node;

  /// How many nodes sit below this step.
  final int inside;

  /// Titles of what this step's work waits on outside every shown step.
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
    return SizedBox(
      width: width,
      child: FieldCard(
        onTap: inside > 0
            ? () => GoRouter.maybeOf(
                context,
              )?.go('/projects/$projectId/roadmap/${node.id}?view=canvas')
            : () => showRoadmapNodeDetail(
                context,
                node,
                inheritedOwner: inheritedOwner,
              ),
        padding: const EdgeInsets.fromLTRB(
          Cobalt.space3,
          Cobalt.space2,
          Cobalt.space2,
          Cobalt.space3,
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          spacing: Cobalt.space1,
          children: [
            Row(
              children: [
                ShadeChip(
                  label: t('roadmap.status.${node.status}'),
                  tone: ShadeChip.fromStatus(node.status),
                ),
                const Spacer(),
                SizedBox.square(
                  dimension: Cobalt.controlHeightDense,
                  child: menu,
                ),
              ],
            ),
            Text(
              node.title,
              style: theme.textTheme.titleSmall,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
            ),
            if (owner != null) line('${t('roadmap.owner')}: $owner'),
            if (inside > 0 && rollup.leaves > 0)
              line(
                t('roadmap.rollupShort', {
                  'done': rollup.done,
                  'leaves': rollup.leaves,
                }),
              ),
            if (inside > 0)
              line(
                inside == 1
                    ? t('roadmap.itemsInsideOne')
                    : t('roadmap.itemsInside', {'n': inside}),
              ),
            if (waitsOn.isNotEmpty)
              line(t('roadmap.waitsOn', {'titles': waitsOn.join(', ')})),
            if (node.deadline != null)
              Padding(
                padding: const EdgeInsets.only(top: Cobalt.space1),
                child: ShadeChip(
                  label: '${isoDate(node.deadline!)}$pressure',
                  tone:
                      node.pressure == 'overdue' || node.pressure == 'high_risk'
                      ? ShadeTone.danger
                      : ShadeTone.neutral,
                ),
              ),
          ],
        ),
      ),
    );
  }
}
