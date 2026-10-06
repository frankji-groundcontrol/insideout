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
    this.progress,
  });

  final String projectId;
  final List<RoadmapNode> nodes;
  final Future<void> Function() onReload;

  /// The server's progress reading. When present it decides Now and Next, so
  /// this view agrees with the Progress tab; without it the status sort below
  /// stands.
  final RoadmapProgress? progress;

  List<RoadmapNode> _band(String Function(RoadmapNode n) bucket, String name) {
    return nodes.where((n) => bucket(n) == name).toList()
      ..sort((a, b) => a.position.compareTo(b.position));
  }

  List<RoadmapNode> _matching(List<ProgressItem> items) {
    final ids = items.map((i) => i.id).toSet();
    return nodes.where((n) => ids.contains(n.id)).toList()
      ..sort((a, b) => a.position.compareTo(b.position));
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
    final placed = progress == null
        ? const <String>{}
        : {...progress!.now, ...progress!.next}.map((i) => i.id).toSet();
    final now = progress == null
        ? _band(bucket, 'now')
        : _matching(progress!.now);
    final next = progress == null
        ? _band(bucket, 'next')
        : _matching(progress!.next);
    final later = progress == null
        ? _band(bucket, 'later')
        : (nodes.where((n) => !placed.contains(n.id)).toList()
            ..sort((a, b) => a.position.compareTo(b.position)));
    return ListView(
      padding: const EdgeInsets.fromLTRB(24, 8, 24, 140),
      children: [
        _BandRow(
          label: l10n.t('roadmap.bandNow'),
          color: palette.surfaceSunken,
          nodes: now,
          allNodes: nodes,
          projectId: projectId,
          onReload: onReload,
        ),
        const SizedBox(height: 16),
        _BandRow(
          label: l10n.t('roadmap.bandNext'),
          color: palette.brandSubtle,
          nodes: next,
          allNodes: nodes,
          projectId: projectId,
          onReload: onReload,
        ),
        const SizedBox(height: 16),
        _BandRow(
          label: l10n.t('roadmap.bandLater'),
          color: palette.brand,
          labelColor: palette.btnFg,
          nodes: later,
          allNodes: nodes,
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
    required this.allNodes,
    required this.projectId,
    required this.onReload,
    this.labelColor,
  });

  final String label;
  final Color color;
  final Color? labelColor;
  final List<RoadmapNode> nodes;

  /// Every node of the project — child progress needs siblings, not just
  /// this band's members.
  final List<RoadmapNode> allNodes;
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
            child: RoadmapNodeCard(
              projectId: projectId,
              node: n,
              onReload: onReload,
              compact: true,
              childProgress: RoadmapNodeCard.childDoneShare(allNodes, n),
            ),
          ),
      ],
    );
  }
}
