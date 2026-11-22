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
        // Someone else's view holds their nodes across the whole project,
        // not only the opened tree.
        if (_view == _RoadmapView.mine && _theirs != null) {
          return FutureBuilder(
            future: _c.projectNodes,
            builder: (context, ps) => ps.hasData
                ? _canvasBody(ps.data!, l10n)
                : readGate(ps, _c.reload),
          );
        }
        return _canvasBody(snap.data!, l10n);
      },
    );
  }

  /// The Canvas over [all]: the steps, or Mine's (or theirs) owned roots.
  Widget _canvasBody(List<RoadmapNode> all, Appearance l10n) {
    // Mine's steps are the topmost nodes the caller, or on someone
    // else's node its person in charge, owns; under an ancestor they
    // own, the whole opened tree is theirs. The steps still read the
    // whole list for links and counts.
    final them = _theirs;
    final uid = them?.userId ?? context.read<Session>().user?.id ?? '';
    final mine = _view == _RoadmapView.mine;
    final shown = mine ? _c.mine(all, uid) : all;
    if (shown.isEmpty) {
      return BlockEmpty(
        icon: Icons.person_outline,
        body: them == null
            ? l10n.t('roadmap.mineEmpty')
            : l10n.t('roadmap.theirsEmpty', {'name': them.firstName}),
      );
    }
    return RoadmapCanvas(
      projectId: widget.projectId,
      steps: mine ? roadmapRoots(shown) : roadmapSteps(all),
      all: all,
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
      // Shared cursors live in the steps' flow; Mine lays out other
      // steps, so it neither sends nor shows them.
      onCursor: mine ? null : _c.sendCursor,
      cursors: mine ? const [] : _c.cursors.values.toList(),
    );
  }

  /// The opened node's person in charge when that is someone else: the
  /// fourth toggle is then theirs.
  RoadmapInCharge? get _theirs {
    final who = _c.glance?.inCharge;
    final uid = context.read<Session>().user?.id ?? '';
    return who == null || who.userId.isEmpty || who.userId == uid ? null : who;
  }

  /// The loaded node [id] for its window, with its inherited owner: from
  /// the shown tree, else the whole project (someone else's view reads it).
  Future<RoadmapWindowNode?> _resolve(String id) async {
    for (final nodes in [_c.nodes, _c.projectNodes]) {
      final all = await nodes.catchError((_) => <RoadmapNode>[]);
      final node = all.where((n) => n.id == id).firstOrNull;
      if (node != null) {
        return (node: node, inheritedOwner: _inherited(node, all));
      }
    }
    return null;
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
