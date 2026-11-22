import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../chrome/time_ago.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_modal.dart';
import '../../theme/block_state.dart';
import '../../theme/cobalt.dart';

/// The read-only node detail: what the node is, who holds it, how far its
/// leaves are, its evidence and who handed it to whom. Every member may read
/// it; nothing here writes.
Future<void> showRoadmapNodeDetail(
  BuildContext context,
  RoadmapNode node, {
  ({String name, String via})? inheritedOwner,
}) => showBlockModal<void>(
  context: context,
  title: node.title,
  body: RoadmapNodeDetailBody(node: node, inheritedOwner: inheritedOwner),
);

/// Who holds [node]: its own owner, else the one it inherits and the
/// ancestor it comes through; null when nobody does.
String? roadmapOwnerLabel(
  String Function(String key, [Map<String, Object?>? args]) t,
  RoadmapNode node,
  ({String name, String via})? inherited,
) {
  if (node.ownerName.isNotEmpty) return node.ownerName;
  if (inherited == null) return null;
  return t('roadmap.ownerVia', {
    'name': inherited.name.isEmpty ? t('roadmap.someone') : inherited.name,
    'via': inherited.via,
  });
}

/// The detail's content, which the detail modal and a node's window show.
class RoadmapNodeDetailBody extends StatefulWidget {
  const RoadmapNodeDetailBody({
    super.key,
    required this.node,
    this.inheritedOwner,
  });

  final RoadmapNode node;
  final ({String name, String via})? inheritedOwner;

  @override
  State<RoadmapNodeDetailBody> createState() => _RoadmapNodeDetailBodyState();
}

class _RoadmapNodeDetailBodyState extends State<RoadmapNodeDetailBody> {
  late Future<List<NodeEvidence>> _evidence;
  late Future<List<NodeAssignment>> _history;

  @override
  void initState() {
    super.initState();
    _load();
  }

  void _load() {
    final api = context.read<Session>().api;
    _evidence = api.listNodeEvidence(widget.node.id);
    _history = api.listNodeAssignments(widget.node.id);
  }

  void _retry() => setState(_load);

  @override
  Widget build(BuildContext context) {
    final node = widget.node;
    final t = context.watch<Appearance>().t;
    final theme = Theme.of(context);
    Widget heading(String key) => Padding(
      padding: const EdgeInsets.only(top: Cobalt.space4, bottom: 4),
      child: Text(t(key), style: theme.textTheme.labelLarge),
    );
    Widget line(String text) => Text(text, style: theme.textTheme.bodyMedium);
    final rollup = node.rollup;
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        line(t('roadmap.status.${node.status}')),
        if (node.description.isNotEmpty) ...[
          heading('roadmap.description'),
          line(node.description),
        ],
        if (node.acceptance.isNotEmpty) ...[
          heading('roadmap.acceptance'),
          line(node.acceptance),
        ],
        heading('roadmap.owner'),
        line(
          roadmapOwnerLabel(t, node, widget.inheritedOwner) ??
              t('roadmap.ownerNone'),
        ),
        if (node.deadline != null) ...[
          heading('roadmap.deadline'),
          line(isoDate(node.deadline!)),
        ],
        if (rollup.leaves > 0) ...[
          const SizedBox(height: Cobalt.space4),
          line(
            t('roadmap.rollup', {
              'done': rollup.done,
              'leaves': rollup.leaves,
              'validated': rollup.validated,
            }),
          ),
        ],
        heading('roadmap.evidence'),
        FutureBuilder(
          future: _evidence,
          builder: (context, snap) {
            if (!snap.hasData) return readGate(snap, _retry);
            if (snap.data!.isEmpty) return line(t('roadmap.evidenceNone'));
            return Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [for (final e in snap.data!) line(e.detail)],
            );
          },
        ),
        heading('roadmap.history'),
        FutureBuilder(
          future: _history,
          builder: (context, snap) {
            if (!snap.hasData) return readGate(snap, _retry);
            if (snap.data!.isEmpty) return line(t('roadmap.historyNone'));
            final someone = t('roadmap.someone');
            return Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                for (final a in snap.data!)
                  line(
                    '${isoDate(a.createdAt)} · '
                    '${t('roadmap.history.${a.kind}', {'by': a.assignedByName ?? someone, 'from': a.fromName ?? someone, 'to': a.toName ?? someone})}'
                    '${a.note.isEmpty ? '' : ' — ${a.note}'}',
                  ),
              ],
            );
          },
        ),
      ],
    );
  }
}
