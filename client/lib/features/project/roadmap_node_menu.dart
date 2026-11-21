part of 'roadmap_node_card.dart';

class _NodeMenu extends StatelessWidget {
  const _NodeMenu({
    required this.projectId,
    required this.node,
    required this.onReload,
    required this.descendants,
    required this.planWriter,
    required this.parentWritable,
  });

  final String projectId;
  final RoadmapNode node;
  final Future<void> Function() onReload;
  final int descendants;
  final bool planWriter;
  final bool parentWritable;

  /// Move, delete and change the owner: a plan writer, or the owner of a
  /// branch above this node — a delegate keeps the root they were handed.
  bool get _restructure => node.canWrite && (planWriter || parentWritable);

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    PopupMenuItem<String> item(String value, String key) =>
        PopupMenuItem(value: value, child: Text(l10n.t(key)));
    return PopupMenuButton<String>(
      onSelected: (value) => _act(context, value),
      // Read items for everyone; write items only where the server's
      // authority rule (canWrite) says they would succeed.
      itemBuilder: (_) => [
        item('open', 'roadmap.openTree'),
        item('details', 'roadmap.details'),
        if (node.canWrite) ...[
          item('edit', 'roadmap.edit'),
          item('child', 'roadmap.addChild'),
          if (_restructure) ...[
            item('up', 'roadmap.moveUp'),
            item('down', 'roadmap.moveDown'),
          ],
          item('expand', 'roadmap.expandAI'),
          // The validation leg lives on leaves: observations name one
          // piece of shipped work, and only a done leaf has an outcome to
          // confirm — a plan writer's call.
          if (descendants == 0) ...[
            item('observe', 'roadmap.observe'),
            if (node.status == 'done' && planWriter)
              item('validate', 'roadmap.validate'),
          ],
          if (_restructure) item('delegate', 'roadmap.delegate'),
        ],
        if (node.ownerId != null &&
            node.ownerId == context.read<Session>().user?.id)
          item('handback', 'roadmap.handBack'),
        if (_restructure) item('delete', 'roadmap.delete'),
      ],
    );
  }

  Future<void> _act(BuildContext context, String value) async {
    final api = context.read<Session>().api;
    final l10n = context.read<Appearance>();
    if (value == 'open') {
      GoRouter.maybeOf(context)?.go('/projects/$projectId/roadmap/${node.id}');
    } else if (value == 'details') {
      await showRoadmapNodeDetail(context, node);
    } else if (value == 'delegate') {
      await _delegate(context, l10n);
    } else if (value == 'handback') {
      await _handBack(context, l10n);
    } else if (value == 'edit') {
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
