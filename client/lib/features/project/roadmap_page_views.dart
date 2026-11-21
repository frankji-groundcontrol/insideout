part of 'roadmap_page.dart';

/// The roadmap page's view bodies: the Progress reading, and the Bands,
/// Canvas and Mine views over the node list.
extension _RoadmapViews on _RoadmapPageState {
  Widget _progressBody(Appearance l10n) {
    return FutureBuilder(
      future: _c.progress,
      builder: (context, snap) {
        final progress = _c.liveProgress ?? snap.data;
        if (progress == null) {
          return readGate(snap, _c.reload);
        }
        // Progress lists only dated work, so an empty one does
        // not mean there are no goals.
        return FutureBuilder(
          future: _c.nodes,
          builder: (context, nodes) => RoadmapProgressView(
            progress: progress,
            // Blank until the goals are known, so the wrong
            // sentence never flashes.
            emptyLabel: nodes.hasData
                ? l10n.t(
                    nodes.data!.isEmpty
                        ? 'roadmap.empty'
                        : 'roadmap.noneScheduled',
                  )
                : '',
          ),
        );
      },
    );
  }

  Widget _treeBody(Appearance l10n) {
    return FutureBuilder(
      future: _c.nodes,
      builder: (context, snap) {
        if (!snap.hasData) {
          return readGate(snap, _c.reload);
        }
        if (snap.data!.isEmpty) {
          return BlockEmpty(
            icon: Icons.account_tree_outlined,
            body: l10n.t('roadmap.empty'),
          );
        }
        if (_view == _RoadmapView.bands) {
          // The reading the Progress tab shows, so a reload or
          // a quiet stream cannot make the two disagree.
          return FutureBuilder(
            future: _c.progress,
            builder: (context, ps) {
              final progress = _c.liveProgress ?? ps.data;
              if (progress == null && !ps.hasError) {
                return const BlockLoading();
              }
              return RoadmapBands(
                projectId: widget.projectId,
                nodes: snap.data!,
                onReload: _c.reload,
                progress: progress,
                planWriter: _c.planWriter,
                card: (n) => _card(n, snap.data!, compact: true),
              );
            },
          );
        }
        final all = snap.data!;
        // Mine lays out only the caller's branches, each topmost owned node
        // a tree; the nodes still read the whole list for parent and count.
        // Under an ancestor the caller owns, the whole opened tree is theirs.
        final uid = context.read<Session>().user?.id ?? '';
        final shown = _view == _RoadmapView.mine ? _c.mine(all, uid) : all;
        if (shown.isEmpty) {
          return BlockEmpty(
            icon: Icons.person_outline,
            body: l10n.t('roadmap.mineEmpty'),
          );
        }
        return RoadmapCanvas(
          projectId: widget.projectId,
          nodes: shown,
          titles: {..._c.outsideTitles, for (final n in all) n.id: n.title},
          inheritedOwner: (n) => _inherited(n, all),
          menu: (n) => RoadmapNodeMenu(
            projectId: widget.projectId,
            node: n,
            onReload: _c.reload,
            descendants: roadmapDescendantCount(all, n),
            planWriter: _c.planWriter,
            parentWritable: roadmapParentWritable(all, n),
            inheritedOwner: _inherited(n, all),
            openedRoot: n.id == widget.rootId,
          ),
          // Shared cursors live in the whole tree's graph; Mine lays out
          // other trees, so it neither sends nor shows them.
          onCursor: shown == all ? _c.sendCursor : null,
          cursors: shown == all ? _c.cursors.values.toList() : const [],
        );
      },
    );
  }

  /// The owner [node] holds through an ancestor, read against every loaded
  /// node [all] and, in an opened tree, the ancestors above it.
  ({String name, String via})? _inherited(
    RoadmapNode node,
    List<RoadmapNode> all,
  ) => roadmapInheritedOwner(
    all,
    node,
    above: _c.opened?.ancestors ?? const [],
    names: _c.names,
  );

  /// One node's card, read against every loaded node [all] and, in an
  /// opened tree, the ancestors above it.
  Widget _card(
    RoadmapNode node,
    List<RoadmapNode> all, {
    bool compact = false,
  }) => RoadmapNodeCard(
    projectId: widget.projectId,
    node: node,
    onReload: _c.reload,
    compact: compact,
    descendants: roadmapDescendantCount(all, node),
    planWriter: _c.planWriter,
    parentWritable: roadmapParentWritable(all, node),
    inheritedOwner: _inherited(node, all),
    openedRoot: node.id == widget.rootId,
  );
}
