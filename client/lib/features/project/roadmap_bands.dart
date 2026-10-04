import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import 'roadmap_band_scaffold.dart';
import 'roadmap_node_card.dart';

/// Lock B — Now / Next / Later shade bands that still hold item cards.
class RoadmapBands extends StatelessWidget {
  const RoadmapBands({
    super.key,
    required this.projectId,
    required this.nodes,
    required this.onReload,
  });

  final String projectId;
  final List<RoadmapNode> nodes;
  final Future<void> Function() onReload;

  List<RoadmapNode> _band(String Function(RoadmapNode n) bucket, String name) {
    return nodes.where((n) => bucket(n) == name).toList()..sort((a, b) => a.position.compareTo(b.position));
  }

  static String bucket(RoadmapNode n) {
    return switch (n.status) {
      'in_progress' || 'done' => 'now',
      'pending' => 'next',
      _ => 'later',
    };
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final palette = Cobalt.of(context);
    final now = _band(bucket, 'now');
    final next = _band(bucket, 'next');
    final later = _band(bucket, 'later');
    return ListView(
      padding: const EdgeInsets.fromLTRB(24, 8, 24, 140),
      children: [
        _BandRow(
          label: l10n.t('roadmap.bandNow'),
          color: palette.surfaceSunken,
          nodes: now,
          projectId: projectId,
          onReload: onReload,
        ),
        const SizedBox(height: 16),
        _BandRow(
          label: l10n.t('roadmap.bandNext'),
          color: palette.brandSubtle,
          nodes: next,
          projectId: projectId,
          onReload: onReload,
        ),
        const SizedBox(height: 16),
        _BandRow(
          label: l10n.t('roadmap.bandLater'),
          color: palette.brand,
          labelColor: palette.btnFg,
          nodes: later,
          projectId: projectId,
          onReload: onReload,
        ),
      ],
    );
  }
}

class _BandRow extends StatelessWidget {
  const _BandRow({
    required this.label,
    required this.color,
    required this.nodes,
    required this.projectId,
    required this.onReload,
    this.labelColor,
  });

  final String label;
  final Color color;
  final Color? labelColor;
  final List<RoadmapNode> nodes;
  final String projectId;
  final Future<void> Function() onReload;

  @override
  Widget build(BuildContext context) {
    return RoadmapBandScaffold(
      label: label,
      color: color,
      labelColor: labelColor,
      itemCards: [
        for (final n in nodes)
          SizedBox(
            width: 240,
            child: RoadmapNodeCard(projectId: projectId, node: n, onReload: onReload, compact: true),
          ),
      ],
    );
  }
}
