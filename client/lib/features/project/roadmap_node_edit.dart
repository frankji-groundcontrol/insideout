part of 'roadmap_node_card.dart';

/// The node edit dialog: title, description, acceptance, owner, status and
/// deadline in one form.
extension _NodeEdit on _NodeMenu {
  Future<void> _edit(BuildContext context, Appearance l10n) async {
    final api = context.read<Session>().api;
    final t = l10n.t;
    final navigator = Navigator.of(context);
    final title = TextEditingController(text: node.title);
    final desc = TextEditingController(text: node.description);
    final acceptance = TextEditingController(text: node.acceptance);
    var status = node.status;
    String? completionReason;
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
            // A delegate keeps the root they were handed: no owner picker,
            // and the unchanged owner they send back is no change.
            if (_restructure) ...[
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
            ],
            BlockSelect<String>(
              key: const Key('status-picker'),
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
              onChanged: (v) {
                pickRoadmapDoneStatus(
                  context: ctx,
                  node: node,
                  current: status,
                  picked: v,
                  loadEvidence: () => api.listNodeEvidence(node.id),
                  t: t,
                  apply: (next, reason) => setLocal(() {
                    status = next;
                    completionReason = reason;
                  }),
                );
              },
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
                      // A deadline is a calendar day. Noon UTC on that day
                      // reads as the picked date wherever the instant is
                      // formatted (isoDate shows the UTC date) and never
                      // lands before the picked day's local midnight.
                      setLocal(
                        () => deadline = DateTime.utc(
                          picked.year,
                          picked.month,
                          picked.day,
                          12,
                        ).toIso8601String(),
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
    // Only what changed goes out (rv5-PARC-06): an untouched owner who has
    // since left the workspace would otherwise fail the whole save.
    String? changed(String now, String before) => now == before ? null : now;
    final edits = (
      title: changed(title.text.trim(), node.title),
      description: changed(desc.text, node.description),
      status: changed(status, node.status),
      deadline: changed(deadline ?? '', node.deadline ?? ''),
      acceptance: changed(acceptance.text.trim(), node.acceptance),
      ownerId: changed(ownerId ?? '', node.ownerId ?? ''),
    );
    final any = [
      edits.title,
      edits.description,
      edits.status,
      edits.deadline,
      edits.acceptance,
      edits.ownerId,
    ].any((v) => v != null);
    if (ok == true && any && context.mounted) {
      final done = await guardWrite(
        context,
        () => api.updateRoadmapNode(
          node.id,
          title: edits.title,
          description: edits.description,
          status: edits.status,
          deadline: edits.deadline,
          acceptance: edits.acceptance,
          ownerId: edits.ownerId,
          completionReason: status == 'done' && node.status == 'in_progress'
              ? completionReason
              : null,
        ),
      );
      if (done && context.mounted) await onReload();
    }
    // Not disposed: the pop animation keeps the dialog's TextFields mounted
    // past this await, and the guard's SnackBar rebuilds the overlay — a
    // disposed controller would throw. GC'd once the dialog unmounts.
  }
}
