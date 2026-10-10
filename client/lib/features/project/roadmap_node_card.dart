import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../chrome/time_ago.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/block_field.dart';
import '../../theme/block_form.dart';
import '../../theme/block_modal.dart';
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
      final t = l10n.t;
      final values = await showBlockForm(
        context: context,
        title: t('roadmap.addChild'),
        confirmLabel: t('roadmap.add'),
        fields: [BlockFormField(label: t('roadmap.childPlaceholder'))],
      );
      if (values == null || !context.mounted) return;
      final title = values.first.trim();
      if (title.isEmpty) return;
      final done = await guardWrite(
        context,
        () => api.createRoadmapNode(projectId, title, parentId: node.id),
      );
      if (done && context.mounted) await onReload();
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
      final ok = await showBlockConfirm(
        context: context,
        title: l10n.t('roadmap.delete'),
        message: l10n.t('roadmap.deleteConfirm'),
        confirmText: l10n.t('common.delete'),
        danger: true,
      );
      if (!ok || !context.mounted) return;
      final done = await guardWrite(
        context,
        () => api.deleteRoadmapNode(node.id),
      );
      if (done && context.mounted) await onReload();
    }
  }

  Future<void> _edit(BuildContext context, Appearance l10n) async {
    final api = context.read<Session>().api;
    final t = l10n.t;
    final navigator = Navigator.of(context);
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
    if (!context.mounted) return;
    final ok = await showBlockModal<bool>(
      context: context,
      title: t('roadmap.edit'),
      body: StatefulBuilder(
        builder: (ctx, setLocal) => Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            BlockField(controller: title, label: t('roadmap.rootPlaceholder')),
            const SizedBox(height: Cobalt.space4),
            BlockField(controller: desc, label: t('roadmap.descPlaceholder')),
            const SizedBox(height: Cobalt.space4),
            BlockField(
              key: const Key('acceptance-field'),
              controller: acceptance,
              label: t('roadmap.acceptanceHint'),
            ),
            const SizedBox(height: Cobalt.space4),
            BlockSelect<String?>(
              key: const Key('owner-picker'),
              value: members.any((m) => m.userId == ownerId) ? ownerId : null,
              placeholder: t('roadmap.ownerPick'),
              options: [
                (value: null, label: t('roadmap.ownerNone')),
                for (final m in members) (value: m.userId, label: m.username),
              ],
              onChanged: (v) => setLocal(() => ownerId = v),
            ),
            const SizedBox(height: Cobalt.space4),
            BlockSelect<String>(
              value: status,
              options: [
                for (final s in const [
                  'locked',
                  'pending',
                  'in_progress',
                  'done',
                ])
                  (value: s, label: t('roadmap.status.$s')),
              ],
              onChanged: (v) => setLocal(() => status = v ?? status),
            ),
            const SizedBox(height: Cobalt.space4),
            Row(
              children: [
                BlockButton(
                  // Cleared is '' (the API's remove value), not null.
                  label: deadline == null || deadline!.isEmpty
                      ? t('roadmap.setDeadline')
                      : isoDate(deadline!),
                  icon: Icons.event_outlined,
                  size: BlockButtonSize.sm,
                  variant: BlockButtonVariant.secondary,
                  onPressed: () async {
                    final picked = await showDatePicker(
                      context: ctx,
                      initialDate: DateTime.now().add(const Duration(days: 7)),
                      firstDate: DateTime.now().subtract(
                        const Duration(days: 1),
                      ),
                      lastDate: DateTime.now().add(
                        const Duration(days: 365 * 3),
                      ),
                    );
                    if (picked != null) {
                      setLocal(
                        () => deadline = picked.toUtc().toIso8601String(),
                      );
                    }
                  },
                ),
                if (deadline != null) ...[
                  const SizedBox(width: Cobalt.space2),
                  BlockIconButton(
                    icon: Icons.event_busy,
                    tooltip: t('roadmap.clearDeadline'),
                    onPressed: () => setLocal(() => deadline = ''),
                  ),
                ],
              ],
            ),
          ],
        ),
      ),
      footer: [
        BlockButton(
          label: t('common.cancel'),
          variant: BlockButtonVariant.secondary,
          onPressed: () => navigator.pop(false),
        ),
        BlockButton(
          label: t('roadmap.save'),
          onPressed: () => navigator.pop(true),
        ),
      ],
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
