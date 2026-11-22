import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/cobalt.dart';
import 'roadmap_lens_banner.dart';
import 'roadmap_node_detail.dart';

/// A node window's body: the node as its person in charge sees it (from
/// GET /roadmap/{nid}/glance, a quiet line while it loads or when it
/// fails), then the node's detail as the detail modal shows it.
class RoadmapWindowBody extends StatefulWidget {
  const RoadmapWindowBody({
    super.key,
    required this.node,
    this.inheritedOwner,
    this.onOpen,
  });

  final RoadmapNode node;
  final ({String name, String via})? inheritedOwner;

  /// Opens a Now item of the glance in its own window.
  final void Function(String nodeId)? onOpen;

  @override
  State<RoadmapWindowBody> createState() => _RoadmapWindowBodyState();
}

class _RoadmapWindowBodyState extends State<RoadmapWindowBody> {
  late final Future<RoadmapGlance> _glance;
  late final String _viewer;

  @override
  void initState() {
    super.initState();
    final session = context.read<Session>();
    _glance = session.api.roadmapGlance(widget.node.id);
    _viewer = session.user?.id ?? '';
  }

  @override
  Widget build(BuildContext context) {
    final t = context.watch<Appearance>().t;
    final open = widget.onOpen;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        FutureBuilder(
          future: _glance,
          builder: (context, snap) => snap.hasData
              ? RoadmapLensBody(
                  glance: snap.data!,
                  viewerId: _viewer,
                  full: true,
                  onOpen: open == null ? null : (item) => open(item.id),
                )
              : Text(
                  t(
                    snap.hasError
                        ? 'roadmap.window.glanceFailed'
                        : 'roadmap.window.glanceLoading',
                  ),
                  key: const Key('roadmap-window-glance-quiet'),
                  style: Theme.of(context).textTheme.bodySmall,
                ),
        ),
        Divider(
          height: Cobalt.space6,
          thickness: Cobalt.borderContainer,
          color: Cobalt.of(context).strokeSubtle,
        ),
        RoadmapNodeDetailBody(
          node: widget.node,
          inheritedOwner: widget.inheritedOwner,
        ),
      ],
    );
  }
}
