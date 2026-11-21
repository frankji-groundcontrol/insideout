import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../chrome/product_dock.dart';
import '../../session/appearance.dart';
import 'roadmap_band_scaffold.dart';
import 'roadmap_node_card.dart';
import 'roadmap_tree.dart';

/// Lock B — Now / Next / Later bands that still hold item cards, and Done
/// for finished work.
class RoadmapBands extends StatelessWidget {
  const RoadmapBands({
    super.key,
    required this.projectId,
    required this.nodes,
    required this.onReload,
    this.progress,
    this.planWriter = false,
    this.card,
  });

  final String projectId;
  final List<RoadmapNode> nodes;
  final Future<void> Function() onReload;

  /// The server's progress reading. When present it decides Now and Next, so
  /// this view agrees with the Progress tab; without it the status sort below
  /// stands, which like the server keeps Now to in-progress work.
  final RoadmapProgress? progress;

  /// The caller writes the whole plan (from the project view).
  final bool planWriter;

  /// The page's card for a node, which knows the opened tree; without it
  /// each band builds a plain card.
  final Widget Function(RoadmapNode node)? card;

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
      'in_progress' => 'now',
      'pending' => 'next',
      'done' || 'validated' => 'done',
      _ => 'later',
    };
  }

  static bool _finished(RoadmapNode n) =>
      n.status == 'done' || n.status == 'validated';

  /// Later's cards under each top-level branch: the roots, or the children
  /// of a lone root (a built project has one). A node above that level (the
  /// lone root itself) leads, unheaded.
  List<(String?, List<RoadmapNode>)> _laterGroups(List<RoadmapNode> later) {
    final roots = roadmapRoots(nodes);
    final top = roots.length == 1
        ? (nodes.where((n) => n.parentId == roots.single.id).toList()
            ..sort((a, b) => a.position.compareTo(b.position)))
        : roots;
    final byId = {for (final n in nodes) n.id: n};
    final topIds = {for (final t in top) t.id};
    String? branchOf(RoadmapNode n) {
      for (RoadmapNode? p = n; p != null; p = byId[p.parentId]) {
        if (topIds.contains(p.id)) return p.id;
      }
      return null;
    }

    return [
      (null, later.where((n) => branchOf(n) == null).toList()),
      for (final t in top)
        (t.title, later.where((n) => branchOf(n) == t.id).toList()),
    ];
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final now = progress == null
        ? _band(bucket, 'now')
        : _matching(progress!.now);
    final next = progress == null
        ? _band(bucket, 'next')
        : _matching(progress!.next);
    final placed = {...now, ...next}.map((n) => n.id).toSet();
    final rest = nodes.where((n) => !placed.contains(n.id)).toList()
      ..sort((a, b) => a.position.compareTo(b.position));
    final done = rest.where(_finished).toList();
    final later = rest.where((n) => !_finished(n)).toList();
    Widget band(
      String key,
      List<(String?, List<RoadmapNode>)> groups, {
      bool now = false,
    }) => _BandRow(
      label: l10n.t(key),
      now: now,
      groups: groups,
      allNodes: nodes,
      projectId: projectId,
      onReload: onReload,
      planWriter: planWriter,
      card: card,
    );
    return ListView(
      padding: const EdgeInsets.fromLTRB(24, 8, 24, ProductDock.clearance),
      children: [
        band('roadmap.bandNow', [(null, now)], now: true),
        const SizedBox(height: 16),
        band('roadmap.bandNext', [(null, next)]),
        const SizedBox(height: 16),
        band('roadmap.bandLater', _laterGroups(later)),
        if (done.isNotEmpty) ...[
          const SizedBox(height: 16),
          band('roadmap.bandDone', [(null, done)]),
        ],
      ],
    );
  }
}

class _BandRow extends StatelessWidget {
  const _BandRow({
    required this.label,
    required this.now,
    required this.groups,
    required this.allNodes,
    required this.projectId,
    required this.onReload,
    required this.planWriter,
    required this.card,
  });

  final String label;
  final bool now;
  final List<(String?, List<RoadmapNode>)> groups;

  /// Every node of the project — a card's descendants and parent sit
  /// outside this band's members.
  final List<RoadmapNode> allNodes;
  final String projectId;
  final Future<void> Function() onReload;
  final bool planWriter;
  final Widget Function(RoadmapNode node)? card;

  Widget _card(RoadmapNode n) =>
      card?.call(n) ??
      RoadmapNodeCard(
        projectId: projectId,
        node: n,
        onReload: onReload,
        compact: true,
        descendants: roadmapDescendantCount(allNodes, n),
        planWriter: planWriter,
        parentWritable: roadmapParentWritable(allNodes, n),
      );

  @override
  Widget build(BuildContext context) {
    return RoadmapBandScaffold.grouped(
      label: label,
      now: now,
      groups: [
        for (final (heading, nodes) in groups)
          RoadmapBandGroup(heading, [for (final n in nodes) _card(n)]),
      ],
    );
  }
}
