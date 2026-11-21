part of 'roadmap_node_card.dart';

class _NodeMenu extends StatelessWidget {
  const _NodeMenu({
    required this.projectId,
    required this.node,
    required this.onReload,
    required this.descendants,
  });

  final String projectId;
  final RoadmapNode node;
  final Future<void> Function() onReload;
  final int descendants;

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
        // The validation leg lives on leaves: observations name one
        // piece of shipped work, and only a done leaf has an outcome to
        // confirm.
        if (descendants == 0) ...[
          PopupMenuItem(
            value: 'observe',
            child: Text(l10n.t('roadmap.observe')),
          ),
          if (node.status == 'done')
            PopupMenuItem(
              value: 'validate',
              child: Text(l10n.t('roadmap.validate')),
            ),
        ],
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
        fields: [
          BlockFormField(label: t('roadmap.childPlaceholder'), required: true),
        ],
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
      await _expandWithAI(context, node, onReload);
    } else if (value == 'observe') {
      await _observe(context, l10n);
    } else if (value == 'validate') {
      await _validate(context, l10n);
    } else if (value == 'delete') {
      final ok = await showBlockConfirm(
        context: context,
        title: l10n.t('roadmap.delete'),
        message: descendants == 0
            ? l10n.t('roadmap.deleteConfirm')
            : l10n.t('roadmap.deleteConfirmMany', {'n': descendants + 1}),
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

  /// The validation observation form: what was observed, its source, and
  /// the date it was seen. Typed by a human — nothing invites users or
  /// collects metrics.
  Future<void> _observe(BuildContext context, Appearance l10n) async {
    final api = context.read<Session>().api;
    final t = l10n.t;
    final today = DateTime.now();
    final values = await showBlockForm(
      context: context,
      title: t('roadmap.observe'),
      confirmLabel: t('roadmap.observeSave'),
      fields: [
        BlockFormField(
          label: t('roadmap.observeHint'),
          multiline: true,
          required: true,
        ),
        BlockFormField(label: t('roadmap.observeSourceHint')),
        // The field carries the date, not a picker: one fewer surface, and
        // the server states the format it wants.
        BlockFormField(
          label: t('roadmap.observeDate'),
          initial:
              '${today.year.toString().padLeft(4, '0')}-'
              '${today.month.toString().padLeft(2, '0')}-'
              '${today.day.toString().padLeft(2, '0')}',
        ),
      ],
    );
    if (values == null || !context.mounted) return;
    final detail = values[0].trim();
    if (detail.isEmpty) return;
    // An unparsable date falls back to today: the field was prefilled
    // with it, so clearing it reads as "seen today".
    final observedAt = DateTime.tryParse(values[2].trim()) ?? today;
    final done = await guardWrite(
      context,
      () => api.addValidationObservation(
        node.id,
        detail: detail,
        source: values[1].trim(),
        observedAt: observedAt,
      ),
    );
    if (done && context.mounted) await onReload();
  }

  /// The human confirm into validated: the reason is the form's one
  /// required field — a confirm that says nothing is refused.
  Future<void> _validate(BuildContext context, Appearance l10n) async {
    final api = context.read<Session>().api;
    final t = l10n.t;
    final values = await showBlockForm(
      context: context,
      title: t('roadmap.validate'),
      confirmLabel: t('common.confirm'),
      fields: [
        BlockFormField(
          label: t('roadmap.validateHint'),
          multiline: true,
          required: true,
        ),
      ],
    );
    if (values == null || !context.mounted) return;
    final reason = values.first.trim();
    if (reason.isEmpty) return;
    final done = await guardWrite(
      context,
      () => api.validateRoadmapNode(node.id, reason),
    );
    if (done && context.mounted) await onReload();
  }
}
