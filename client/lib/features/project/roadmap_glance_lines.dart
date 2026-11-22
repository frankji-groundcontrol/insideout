import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';

/// A first glance at a node, as My work and the opened node's banner show
/// it: its Now items with their pressure, then what it waits on and what it
/// blocks, each line hidden when empty. [onOpen], when set, opens a Now item.
class RoadmapGlanceLines extends StatelessWidget {
  const RoadmapGlanceLines({
    super.key,
    required this.now,
    required this.waitsOn,
    required this.blocking,
    this.onOpen,
  });

  final List<ProgressItem> now;
  final List<RoadmapLink> waitsOn;
  final List<RoadmapLink> blocking;
  final void Function(ProgressItem item)? onOpen;

  @override
  Widget build(BuildContext context) {
    final t = context.watch<Appearance>().t;
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (final item in now)
          _NowRow(item, onTap: onOpen == null ? null : () => onOpen!(item)),
        if (waitsOn.isNotEmpty)
          _LinkLine(t('roadmap.waitsOn', {'titles': _names(waitsOn)})),
        if (blocking.isNotEmpty)
          _LinkLine(t('home.myWork.blocking', {'titles': _names(blocking)})),
      ],
    );
  }
}

String _names(List<RoadmapLink> links) => [
  for (final l in links)
    l.ownerName.isEmpty ? l.title : '${l.title} (${l.ownerName})',
].join(', ');

class _NowRow extends StatelessWidget {
  const _NowRow(this.item, {this.onTap});

  final ProgressItem item;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final pressure = item.pressure;
    final row = Padding(
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
    return onTap == null ? row : InkWell(onTap: onTap, child: row);
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
