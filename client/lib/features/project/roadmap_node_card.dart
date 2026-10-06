import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../chrome/time_ago.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';
import '../../theme/field_card.dart';
import '../../widgets/guarded_action.dart';

class RoadmapNodeCard extends StatelessWidget {
  const RoadmapNodeCard({
    super.key,
    required this.projectId,
    required this.node,
    required this.onReload,
    this.compact = false,
    this.childProgress,
  });

  final String projectId;
  final RoadmapNode node;
  final Future<void> Function() onReload;
  final bool compact;

  /// Share of this node's children that are done (0..1), or null for a
  /// leaf — leaves show no bar rather than an invented percentage. Only a
  /// caller holding the whole tree can compute it.
  final double? childProgress;

  /// done-children share of [n] within [all], or null when [n] is a leaf.
  static double? childDoneShare(List<RoadmapNode> all, RoadmapNode n) {
    final children = all.where((o) => o.parentId == n.id).toList();
    if (children.isEmpty) return null;
    final done = children.where((c) => c.status == 'done').length;
    return done / children.length;
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    return FieldCard(
      padding: const EdgeInsets.fromLTRB(16, 14, 8, 14),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              ShadeChip(
                label: l10n.t('roadmap.status.${node.status}'),
                tone: ShadeChip.fromStatus(node.status),
              ),
              const Spacer(),
              _NodeMenu(projectId: projectId, node: node, onReload: onReload),
            ],
          ),
          const SizedBox(height: 10),
          Text(node.title, style: theme.textTheme.titleMedium),
          if (node.editorName != null)
            Text(node.editorName!, style: theme.textTheme.bodySmall),
          if (node.deadline != null)
            Padding(
              padding: const EdgeInsets.only(top: 6),
              child: ShadeChip(
                label:
                    '${isoDate(node.deadline!)}${node.pressure != null && node.pressure != 'normal' ? ' · ${l10n.t('roadmap.pressure.${node.pressure}')}' : ''}',
                tone: node.pressure == 'overdue' || node.pressure == 'high_risk'
                    ? ShadeTone.danger
                    : ShadeTone.neutral,
              ),
            ),
          if (node.acceptance.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 6),
              child: Text(
                '${l10n.t('roadmap.acceptance')}: ${node.acceptance}',
                style: theme.textTheme.bodySmall,
              ),
            ),
          if (node.ownerName.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 4),
              child: Text(
                '${l10n.t('roadmap.owner')}: ${node.ownerName}',
                style: theme.textTheme.bodySmall,
              ),
            ),
          if (childProgress != null) ...[
            const SizedBox(height: 12),
            ClipRRect(
              borderRadius: BorderRadius.circular(Cobalt.radiusPill),
              child: LinearProgressIndicator(
                value: childProgress,
                minHeight: 4,
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _NodeMenu extends StatelessWidget {
  const _NodeMenu({
    required this.projectId,
    required this.node,
    required this.onReload,
  });

  final String projectId;
  final RoadmapNode node;
  final Future<void> Function() onReload;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return PopupMenuButton<String>(
      onSelected: (value) => _act(context, value),
      itemBuilder: (_) => [
        PopupMenuItem(value: 'edit', child: Text(l10n.t('roadmap.edit'))),
        PopupMenuItem(value: 'child', child: Text(l10n.t('roadmap.addChild'))),
        PopupMenuItem(value: 'up', child: Text(l10n.t('roadmap.moveUp'))),
        PopupMenuItem(value: 'down', child: Text(l10n.t('roadmap.moveDown'))),
        PopupMenuItem(value: 'expand', child: Text(l10n.t('roadmap.expandAI'))),
        PopupMenuItem(value: 'delete', child: Text(l10n.t('roadmap.delete'))),
      ],
    );
  }

  Future<void> _act(BuildContext context, String value) async {
    final api = context.read<Session>().api;
    final l10n = context.read<Appearance>();
    if (value == 'edit') {
      await _edit(context, l10n);
    } else if (value == 'child') {
      final title = TextEditingController();
      final ok = await showDialog<bool>(
        context: context,
        builder: (ctx) => AlertDialog(
          title: Text(l10n.t('roadmap.addChild')),
          content: TextField(
            controller: title,
            decoration: InputDecoration(
              labelText: l10n.t('roadmap.childPlaceholder'),
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: Text(l10n.t('common.cancel')),
            ),
            FilledButton(
              onPressed: () => Navigator.pop(ctx, true),
              child: Text(l10n.t('roadmap.add')),
            ),
          ],
        ),
      );
      if (ok == true && title.text.trim().isNotEmpty && context.mounted) {
        final done = await guardWrite(
          context,
          () => api.createRoadmapNode(
            projectId,
            title.text.trim(),
            parentId: node.id,
          ),
        );
        if (done && context.mounted) await onReload();
      }
      // Not disposed: the pop animation keeps the dialog's TextField mounted
      // past this await, and the guard's SnackBar rebuilds the overlay — a
      // disposed controller would throw. GC'd once the dialog unmounts.
    } else if (value == 'up') {
      final done = await guardWrite(
        context,
        () => api.moveRoadmapNode(
          node.id,
          parentId: node.parentId,
          position: node.position == 0 ? 0 : node.position - 1,
        ),
      );
      if (done && context.mounted) await onReload();
    } else if (value == 'down') {
      final done = await guardWrite(
        context,
        () => api.moveRoadmapNode(
          node.id,
          parentId: node.parentId,
          position: node.position + 1,
        ),
      );
      if (done && context.mounted) await onReload();
    } else if (value == 'expand') {
      final done = await guardWrite(
        context,
        () => api.expandRoadmapNode(node.id),
      );
      if (done && context.mounted) await onReload();
    } else if (value == 'delete') {
      final done = await guardWrite(
        context,
        () => api.deleteRoadmapNode(node.id),
      );
      if (done && context.mounted) await onReload();
    }
  }

  Future<void> _edit(BuildContext context, Appearance l10n) async {
    final api = context.read<Session>().api;
    final title = TextEditingController(text: node.title);
    final desc = TextEditingController(text: node.description);
    final acceptance = TextEditingController(text: node.acceptance);
    var status = node.status;
    String? deadline = node.deadline;
    String? ownerId = node.ownerId;
    List<WorkspaceMember> members = const [];
    try {
      final project = await api.getProject(projectId);
      final wsId = project['workspaceId'] as String?;
      if (wsId != null) members = await api.listMembers(wsId);
    } on Exception {
      // Owner picker stays empty; the rest of the edit still works.
    }
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setLocal) => AlertDialog(
          title: Text(l10n.t('roadmap.edit')),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: title,
                decoration: InputDecoration(
                  labelText: l10n.t('roadmap.rootPlaceholder'),
                ),
              ),
              TextField(
                controller: desc,
                decoration: InputDecoration(
                  labelText: l10n.t('roadmap.descPlaceholder'),
                ),
              ),
              TextField(
                key: const Key('acceptance-field'),
                controller: acceptance,
                decoration: InputDecoration(
                  labelText: l10n.t('roadmap.acceptanceHint'),
                ),
              ),
              DropdownButton<String?>(
                key: const Key('owner-picker'),
                isExpanded: true,
                value: members.any((m) => m.userId == ownerId) ? ownerId : null,
                hint: Text(l10n.t('roadmap.ownerPick')),
                items: [
                  DropdownMenuItem(
                    value: null,
                    child: Text(l10n.t('roadmap.ownerNone')),
                  ),
                  ...members.map(
                    (m) => DropdownMenuItem(
                      value: m.userId,
                      child: Text(m.username),
                    ),
                  ),
                ],
                onChanged: (v) => setLocal(() => ownerId = v),
              ),
              DropdownButton<String>(
                value: status,
                items: ['locked', 'pending', 'in_progress', 'done']
                    .map(
                      (s) => DropdownMenuItem(
                        value: s,
                        child: Text(l10n.t('roadmap.status.$s')),
                      ),
                    )
                    .toList(),
                onChanged: (v) => setLocal(() => status = v ?? status),
              ),
              Row(
                children: [
                  TextButton(
                    onPressed: () async {
                      final picked = await showDatePicker(
                        context: ctx,
                        initialDate: DateTime.now().add(
                          const Duration(days: 7),
                        ),
                        firstDate: DateTime.now().subtract(
                          const Duration(days: 1),
                        ),
                        lastDate: DateTime.now().add(
                          const Duration(days: 365 * 3),
                        ),
                      );
                      if (picked != null)
                        setLocal(
                          () => deadline = picked.toUtc().toIso8601String(),
                        );
                    },
                    child: Text(
                      deadline == null
                          ? l10n.t('roadmap.setDeadline')
                          : isoDate(deadline!),
                    ),
                  ),
                  if (deadline != null)
                    IconButton(
                      tooltip: l10n.t('roadmap.clearDeadline'),
                      onPressed: () => setLocal(() => deadline = ''),
                      icon: const Icon(Icons.event_busy, size: 18),
                    ),
                ],
              ),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: Text(l10n.t('common.cancel')),
            ),
            FilledButton(
              onPressed: () => Navigator.pop(ctx, true),
              child: Text(l10n.t('roadmap.save')),
            ),
          ],
        ),
      ),
    );
    if (ok == true && context.mounted) {
      final done = await guardWrite(
        context,
        () => api.updateRoadmapNode(
          node.id,
          title: title.text.trim(),
          description: desc.text,
          status: status,
          deadline: deadline == '' ? '' : deadline,
          acceptance: acceptance.text.trim(),
          ownerId: ownerId ?? '',
        ),
      );
      if (done && context.mounted) await onReload();
    }
    // Not disposed: the pop animation keeps the dialog's TextFields mounted
    // past this await, and the guard's SnackBar rebuilds the overlay — a
    // disposed controller would throw. GC'd once the dialog unmounts.
  }
}
