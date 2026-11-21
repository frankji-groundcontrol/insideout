import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';
import '../../theme/field_card.dart';

/// "My work" on the signed-in landing: the caller's topmost owned roadmap
/// nodes across their workspaces (GET /workspaces/{id}/roadmap/mine), each
/// opening its own tree. Hidden while loading and when there is none.
class MyWorkSection extends StatefulWidget {
  const MyWorkSection({super.key, required this.workspaces});

  final List<Workspace> workspaces;

  @override
  State<MyWorkSection> createState() => _MyWorkSectionState();
}

class _MyWorkSectionState extends State<MyWorkSection> {
  late final Future<List<RoadmapNodeRead>> _future;

  @override
  void initState() {
    super.initState();
    final api = context.read<Session>().api;
    _future = Future.wait([
      // One unreadable workspace leaves the others' items on the landing.
      for (final w in widget.workspaces)
        api
            .listMyRoadmap(w.id)
            .catchError((Object _) => const <RoadmapNodeRead>[]),
    ]).then((lists) => [for (final l in lists) ...l]);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    return FutureBuilder(
      future: _future,
      builder: (context, snap) {
        final items = snap.data ?? const <RoadmapNodeRead>[];
        if (items.isEmpty) return const SizedBox.shrink();
        return Padding(
          padding: const EdgeInsets.only(bottom: Cobalt.space8),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(l10n.t('home.myWork'), style: theme.textTheme.titleLarge),
              for (final e in items) ...[
                const SizedBox(height: Cobalt.space3),
                FieldCard(
                  padding: const EdgeInsets.all(Cobalt.space4),
                  onTap: () => context.go(
                    '/projects/${e.projectId}/roadmap/${e.node.id}',
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Expanded(
                            child: Text(
                              e.node.title,
                              style: theme.textTheme.titleMedium,
                            ),
                          ),
                          ShadeChip(
                            label: l10n.t('roadmap.status.${e.node.status}'),
                            tone: ShadeChip.fromStatus(e.node.status),
                          ),
                        ],
                      ),
                      const SizedBox(height: Cobalt.space2),
                      Text(
                        [
                          e.projectName,
                          ...e.ancestors.map((a) => a.title),
                        ].join(' › '),
                        style: theme.textTheme.bodySmall,
                      ),
                      if (e.node.rollup.leaves > 0)
                        Text(
                          l10n.t('roadmap.rollup', {
                            'done': e.node.rollup.done,
                            'leaves': e.node.rollup.leaves,
                            'validated': e.node.rollup.validated,
                          }),
                          style: theme.textTheme.bodySmall,
                        ),
                    ],
                  ),
                ),
              ],
            ],
          ),
        );
      },
    );
  }
}
