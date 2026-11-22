import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/cobalt.dart';
import 'my_work_root_card.dart';

/// "My work" on the signed-in landing: every root the caller is in charge of
/// across their workspaces (one GET /roadmap/mine), the ones with overdue or
/// high-risk Now work first. Hidden while loading and when there is none.
class MyWorkSection extends StatefulWidget {
  const MyWorkSection({super.key});

  @override
  State<MyWorkSection> createState() => _MyWorkSectionState();
}

class _MyWorkSectionState extends State<MyWorkSection> {
  late final Future<List<MyRoadmapRoot>> _future;

  @override
  void initState() {
    super.initState();
    // A failed read leaves the rest of the landing as it is.
    _future = context
        .read<Session>()
        .api
        .listMyRoadmap()
        .then(_pressedFirst)
        .catchError((Object _) => const <MyRoadmapRoot>[]);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return FutureBuilder(
      future: _future,
      builder: (context, snap) {
        final roots = snap.data ?? const <MyRoadmapRoot>[];
        if (roots.isEmpty) return const SizedBox.shrink();
        return Padding(
          padding: const EdgeInsets.only(bottom: Cobalt.space8),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                l10n.t('home.myWork'),
                style: Theme.of(context).textTheme.titleLarge,
              ),
              for (final root in roots) ...[
                const SizedBox(height: Cobalt.space3),
                MyWorkRootCard(root),
              ],
            ],
          ),
        );
      },
    );
  }
}

/// The roots with pressed Now work first, each group in the server's order.
List<MyRoadmapRoot> _pressedFirst(List<MyRoadmapRoot> roots) => [
  ...roots.where((r) => r.pressed),
  ...roots.where((r) => !r.pressed),
];
