part of 'roadmap_node_card.dart';

/// "Break down with AI". The AI only drafts: the server's expand route writes
/// nothing and answers `{children: [...]}`. The person reads the draft here and
/// only confirming sends it to the confirm route, the one call that creates the
/// sub-tasks. The web client once read the draft as the created nodes and never
/// confirmed, so the action failed every time (rv4-FULL-01).
Future<void> _expandWithAI(
  BuildContext context,
  RoadmapNode node,
  Future<void> Function() onReload,
) async {
  final api = context.read<Session>().api;
  final t = context.read<Appearance>().t;
  var draft = const <DraftSubtask>[];
  final drafted = await guardWrite(context, () async {
    draft = await api.expandRoadmapNode(node.id);
  });
  if (!drafted || !context.mounted) return;
  if (draft.isEmpty) {
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text(t('roadmap.expandEmpty'))));
    return;
  }

  final theme = Theme.of(context);
  final navigator = Navigator.of(context);
  final confirmed = await showBlockModal<bool>(
    context: context,
    title: t('roadmap.expandReview'),
    body: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (final c in draft)
          Padding(
            padding: const EdgeInsets.only(bottom: Cobalt.space3),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(c.title, style: theme.textTheme.titleSmall),
                if (c.description.isNotEmpty)
                  Text(c.description, style: theme.textTheme.bodySmall),
              ],
            ),
          ),
      ],
    ),
    footer: [
      BlockButton(
        label: t('common.cancel'),
        variant: BlockButtonVariant.secondary,
        onPressed: () => navigator.pop(false),
      ),
      BlockButton(
        label: t('roadmap.expandConfirm', {'n': draft.length}),
        onPressed: () => navigator.pop(true),
      ),
    ],
  );
  if (confirmed != true || !context.mounted) return;
  final done = await guardWrite(
    context,
    () => api.confirmExpandRoadmapNode(node.id, draft),
  );
  if (done && context.mounted) await onReload();
}
