part of 'roadmap_node_card.dart';

/// Delegate and hand back: the owner changes on the record (an assignment
/// row and a feed update), not through a silent edit.
extension _NodeDelegate on _NodeMenu {
  Future<void> _delegate(BuildContext context, Appearance l10n) async {
    final api = context.read<Session>().api;
    final t = l10n.t;
    final navigator = Navigator.of(context);
    List<WorkspaceMember> members = const [];
    try {
      final project = await api.getProject(projectId);
      final wsId = project['workspaceId'] as String?;
      if (wsId != null) members = await api.listMembers(wsId);
    } on Exception {
      // An empty picker: the dialog says nothing can be chosen.
    }
    if (!context.mounted) return;
    String? to;
    final note = TextEditingController();
    final ok = await showBlockModal<bool>(
      context: context,
      title: t('roadmap.delegate'),
      body: StatefulBuilder(
        builder: (ctx, setLocal) => Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            BlockSelect<String?>(
              key: const Key('delegate-picker'),
              value: to,
              placeholder: t('roadmap.delegatePick'),
              options: [
                for (final m in members)
                  if (m.userId != node.ownerId)
                    (value: m.userId, label: m.username),
              ],
              onChanged: (v) => setLocal(() => to = v),
            ),
            const SizedBox(height: Cobalt.space4),
            BlockField(
              key: const Key('delegate-note'),
              controller: note,
              label: t('roadmap.assignNote'),
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
          label: t('roadmap.delegateConfirm'),
          onPressed: () => navigator.pop(to != null),
        ),
      ],
    );
    final target = to;
    if (ok != true || target == null || !context.mounted) return;
    final done = await guardWrite(
      context,
      () => api.delegateRoadmapNode(node.id, target, note: note.text.trim()),
    );
    if (done && context.mounted) await onReload();
    // The note controller is left to GC, as in the edit dialog: the pop
    // animation still holds its field.
  }

  Future<void> _handBack(BuildContext context, Appearance l10n) async {
    final t = l10n.t;
    final values = await showBlockForm(
      context: context,
      title: t('roadmap.handBack'),
      confirmLabel: t('roadmap.handBack'),
      fields: [BlockFormField(label: t('roadmap.assignNote'))],
    );
    if (values == null || !context.mounted) return;
    final api = context.read<Session>().api;
    final done = await guardWrite(
      context,
      () => api.handBackRoadmapNode(node.id, note: values.first.trim()),
    );
    if (done && context.mounted) await onReload();
  }
}
