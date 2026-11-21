part of 'roadmap_page.dart';

/// The roadmap page's view bodies: the Progress reading, and the Bands and
/// Canvas views over the node list.
extension _RoadmapViews on _RoadmapPageState {
  Widget _progressBody(Appearance l10n) {
    return FutureBuilder(
      future: _progressFuture,
      builder: (context, snap) {
        final progress = _liveProgress ?? snap.data;
        if (progress == null) {
          return readGate(snap, _reload);
        }
        // Progress lists only dated work, so an empty one does
        // not mean there are no goals.
        return FutureBuilder(
          future: _future,
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
      future: _future,
      builder: (context, snap) {
        if (!snap.hasData) {
          return readGate(snap, _reload);
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
            future: _progressFuture,
            builder: (context, ps) {
              final progress = _liveProgress ?? ps.data;
              if (progress == null && !ps.hasError) {
                return const BlockLoading();
              }
              return RoadmapBands(
                projectId: widget.projectId,
                nodes: snap.data!,
                onReload: _reload,
                progress: progress,
                planWriter: _planWriter,
              );
            },
          );
        }
        return Column(
          children: [
            RoadmapMinimap(nodes: snap.data!, onTapBand: _scrollToBand),
            const Divider(height: 1),
            Expanded(
              child: RoadmapCanvas(
                nodes: snap.data!,
                tileBuilder: (node, _) => RoadmapNodeCard(
                  projectId: widget.projectId,
                  node: node,
                  onReload: _reload,
                  childProgress: RoadmapNodeCard.childDoneShare(
                    snap.data!,
                    node,
                  ),
                  descendants: roadmapDescendantCount(snap.data!, node),
                  planWriter: _planWriter,
                  parentWritable: roadmapParentWritable(snap.data!, node),
                ),
                controller: _canvasController,
                onCursor: _sendCursor,
                cursors: _cursors.values.toList(),
              ),
            ),
          ],
        );
      },
    );
  }
}
