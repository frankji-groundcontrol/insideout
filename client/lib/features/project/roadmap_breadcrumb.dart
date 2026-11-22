import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../api/models.dart';
import '../../theme/cobalt.dart';

/// The path from the project's whole tree down to the opened node: the
/// project and each ancestor link back up; the node itself is plain text.
class RoadmapBreadcrumb extends StatelessWidget {
  const RoadmapBreadcrumb({
    super.key,
    required this.projectId,
    required this.read,
    this.onAncestor,
  });

  final String projectId;
  final Future<RoadmapNodeRead> read;

  /// In a node's window an ancestor re-roots the window, not the page.
  final void Function(String nodeId)? onAncestor;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    return FutureBuilder(
      future: read,
      builder: (context, snap) {
        final r = snap.data;
        if (r == null) return const SizedBox.shrink();
        Widget link(String label, String path, [String? id]) => InkWell(
          onTap: () => id != null && onAncestor != null
              ? onAncestor!(id)
              : GoRouter.maybeOf(context)?.go(path),
          child: Text(
            label,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: palette.brand,
              decoration: TextDecoration.underline,
            ),
          ),
        );
        const sep = Text(' / ');
        return Padding(
          padding: const EdgeInsets.only(bottom: Cobalt.space2),
          child: Wrap(
            key: const Key('roadmap-breadcrumb'),
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              link(r.projectName, '/projects/$projectId/roadmap'),
              for (final a in r.ancestors) ...[
                sep,
                link(a.title, '/projects/$projectId/roadmap/${a.id}', a.id),
              ],
              sep,
              Text(r.node.title, style: theme.textTheme.bodyMedium),
            ],
          ),
        );
      },
    );
  }
}
