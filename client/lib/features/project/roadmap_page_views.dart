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
        // a band; the cards still read the whole list for parent and count.
        // Under an ancestor the caller owns, the whole opened tree is theirs.
        final uid = context.read<Session>().user?.id ?? '';
        final shown = _view == _RoadmapView.mine ? _c.mine(all, uid) : all;
        if (shown.isEmpty) {
          return BlockEmpty(
            icon: Icons.person_outline,
            body: l10n.t('roadmap.mineEmpty'),
          );
        }
        // The minimap jumps between bands, so one band needs none.
        final bands = roadmapRoots(shown).length;
        return Column(
          children: [
            if (bands > 1)
              RoadmapMinimap(nodes: shown, onTapBand: _scrollToBand),
            Expanded(
              child: RoadmapCanvas(
                nodes: shown,
                tileBuilder: (node, _) => _card(node, all),
                controller: _canvasController,
                // Shared cursors live in the whole canvas's layout; Mine
                // lays out other bands, so it neither sends nor shows them.
                onCursor: shown == all ? _c.sendCursor : null,
                cursors: shown == all ? _c.cursors.values.toList() : const [],
                collapsed: _collapsed,
                onToggle: _toggleFold,
              ),
            ),
          ],
        );
      },
    );
  }

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
    inheritedOwner: roadmapInheritedOwner(
      all,
      node,
      above: _c.opened?.ancestors ?? const [],
      names: _c.names,
    ),
    openedRoot: node.id == widget.rootId,
  );
}
