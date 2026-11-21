import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_field.dart';
import '../../theme/block_icon_button.dart';
import '../../theme/block_modal.dart';
import '../../theme/cobalt.dart';
import '../../theme/field_card.dart';
import '../../widgets/guarded_action.dart';

/// "Depends on…": what [node] waits on, each link removable, and a search
/// over the project's other nodes to add one. One pick is one write; a
/// refused link (a loop, or a node above or below this one) reads as its
/// reason through guardWrite.
Future<void> editRoadmapDependencies(
  BuildContext context, {
  required String projectId,
  required RoadmapNode node,
  required Future<void> Function() onReload,
}) async {
  final api = context.read<Session>().api;
  final t = context.read<Appearance>().t;
  // The whole project, so a node in an opened tree can wait on work outside.
  var all = const <RoadmapNode>[];
  try {
    all = await api.listRoadmap(projectId);
  } on Exception {
    // An empty picker: it says nothing matches.
  }
  if (!context.mounted) return;
  final pick = await showBlockModal<({String id, bool remove})>(
    context: context,
    title: t('roadmap.dependsOn'),
    body: _Picker(node: node, all: all),
  );
  if (pick == null || !context.mounted) return;
  final done = await guardWrite(
    context,
    () => pick.remove
        ? api.removeRoadmapDependency(node.id, pick.id)
        : api.addRoadmapDependency(node.id, pick.id),
  );
  if (done && context.mounted) await onReload();
}

class _Picker extends StatefulWidget {
  const _Picker({required this.node, required this.all});

  final RoadmapNode node;
  final List<RoadmapNode> all;

  @override
  State<_Picker> createState() => _PickerState();
}

class _PickerState extends State<_Picker> {
  var _query = '';

  /// The node, its ancestors and its descendants: the tree already orders
  /// those, so the server refuses them.
  late final Set<String> _related = () {
    final byId = {for (final n in widget.all) n.id: n};
    Iterable<String> up(String? id) sync* {
      for (var p = byId[id]; p != null; p = byId[p.parentId]) {
        yield p.id;
      }
    }

    return {
      widget.node.id,
      ...up(widget.node.parentId),
      for (final n in widget.all)
        if (up(n.parentId).contains(widget.node.id)) n.id,
    };
  }();

  @override
  Widget build(BuildContext context) {
    final t = context.watch<Appearance>().t;
    final theme = Theme.of(context);
    final titles = {for (final n in widget.all) n.id: n.title};
    final q = _query.trim().toLowerCase();
    final options = [
      for (final n in widget.all)
        if (!_related.contains(n.id) &&
            !widget.node.dependsOn.contains(n.id) &&
            n.title.toLowerCase().contains(q))
          n,
    ];
    void pop(String id, {bool remove = false}) =>
        Navigator.of(context).pop((id: id, remove: remove));
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      spacing: Cobalt.space2,
      children: [
        Text(t('roadmap.waitsOnNow'), style: theme.textTheme.labelMedium),
        if (widget.node.dependsOn.isEmpty)
          Text(t('roadmap.waitsOnNothing'), style: theme.textTheme.bodySmall),
        for (final id in widget.node.dependsOn)
          Row(
            children: [
              Expanded(child: Text(titles[id] ?? t('roadmap.someItem'))),
              BlockIconButton(
                icon: Icons.close,
                semanticLabel: t('roadmap.dependencyRemove', {
                  'title': titles[id] ?? t('roadmap.someItem'),
                }),
                onPressed: () => pop(id, remove: true),
              ),
            ],
          ),
        const SizedBox(height: Cobalt.space2),
        BlockField(
          label: t('roadmap.dependencySearch'),
          autofocus: true,
          onChanged: (v) => setState(() => _query = v),
        ),
        if (options.isEmpty)
          Text(t('roadmap.dependencyNone'), style: theme.textTheme.bodySmall),
        // shortcut: the first matches only; a project past a few dozen nodes
        // wants a scrolled list here.
        for (final n in options.take(12))
          FieldCard(
            key: Key('dependency-option-${n.id}'),
            padding: const EdgeInsets.all(Cobalt.space3),
            onTap: () => pop(n.id),
            child: Text(n.title, maxLines: 1, overflow: TextOverflow.ellipsis),
          ),
      ],
    );
  }
}
