import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/client.dart';
import '../../api/models.dart';
import '../../chrome/time_ago.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_form.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';
import '../../theme/field_card.dart';
import '../../widgets/guarded_action.dart';

/// The Evidence page's updates timeline: the agent-proposal decision loop
/// (Accept/Reject) and the note edit/delete menu. The page supplies the
/// note write verbs ([onEdit], [onRemove]) and reloads through [onChanged].
class ProjectUpdatesFeed extends StatelessWidget {
  const ProjectUpdatesFeed({
    super.key,
    required this.projectId,
    required this.updates,
    required this.onChanged,
    required this.onEdit,
    required this.onRemove,
  });

  final String projectId;
  final List<ProjectUpdate> updates;
  final Future<void> Function() onChanged;
  final Future<void> Function(ProjectUpdate update, String content) onEdit;
  final Future<void> Function(ProjectUpdate update) onRemove;

  // FEAT-03: the proposal decision loop — the server resolves authority
  // (project owner or workspace admin); the web just surfaces the outcome.
  Future<void> _decide(
    BuildContext context,
    ProjectUpdate update, {
    required bool accept,
  }) async {
    final api = context.read<Session>().api;
    final done = await guardWrite(context, () async {
      await api.decideProposal(update.id, accept: accept);
      if (accept) await _applyAcceptedItems(api, update);
    });
    if (done && context.mounted) await onChanged();
  }

  // Accepting a proposal changes the roadmap: each add_node item becomes a
  // node. parentHint matches an existing node's title; empty means a root.
  Future<void> _applyAcceptedItems(ApiClient api, ProjectUpdate update) async {
    final nodes = await api.listRoadmap(projectId);
    for (final item in update.items) {
      if (item.action != 'add_node' || item.title.trim().isEmpty) continue;
      String? parentId;
      final hint = item.parentHint.trim();
      if (hint.isNotEmpty) {
        for (final node in nodes) {
          if (node.title == hint) {
            parentId = node.id;
            break;
          }
        }
      }
      final created = await api.createRoadmapNode(
        projectId,
        item.title.trim(),
        parentId: parentId,
      );
      nodes.add(created);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(l10n.t('project.updates'), style: theme.textTheme.titleLarge),
        const SizedBox(height: 12),
        if (updates.isEmpty)
          Text(l10n.t('github.empty'), style: theme.textTheme.bodyLarge),
        for (final u in updates) ...[
          FieldCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                ShadeChip(
                  label: l10n.t('project.updateKind.${u.kind}'),
                  tone: ShadeTone.neutral,
                ),
                const SizedBox(height: 8),
                Text(u.content, style: theme.textTheme.titleMedium),
                Text(
                  timeAgo(u.createdAt, l10n.t),
                  style: theme.textTheme.bodySmall,
                ),
                Align(
                  alignment: Alignment.centerRight,
                  // FEAT-03: an agent proposal is decided, not
                  // edited — Accept/Reject own the record.
                  child: u.kind == 'agent_proposal' && u.decision.isEmpty
                      ? Wrap(
                          spacing: Cobalt.space2,
                          children: [
                            BlockButton(
                              label: l10n.t('proposal.accept'),
                              size: BlockButtonSize.sm,
                              onPressed: () =>
                                  _decide(context, u, accept: true),
                            ),
                            BlockButton(
                              label: l10n.t('proposal.reject'),
                              size: BlockButtonSize.sm,
                              variant: BlockButtonVariant.secondary,
                              onPressed: () =>
                                  _decide(context, u, accept: false),
                            ),
                          ],
                        )
                      : PopupMenuButton<String>(
                          onSelected: (value) async {
                            if (value == 'edit') {
                              // The form state owns its controller
                              // and disposes it on unmount — no
                              // controller outlives the pop.
                              final values = await showBlockForm(
                                context: context,
                                title: l10n.t('common.edit'),
                                confirmLabel: l10n.t('common.save'),
                                fields: [
                                  BlockFormField(
                                    label: l10n.t('project.updatePlaceholder'),
                                    initial: u.content,
                                  ),
                                ],
                              );
                              if (values != null && context.mounted) {
                                final done = await guardWrite(
                                  context,
                                  () => onEdit(u, values.first.trim()),
                                );
                                if (done && context.mounted) {
                                  await onChanged();
                                }
                              }
                            } else if (value == 'delete') {
                              final done = await guardWrite(
                                context,
                                () => onRemove(u),
                              );
                              if (done && context.mounted) {
                                await onChanged();
                              }
                            }
                          },
                          itemBuilder: (_) => [
                            PopupMenuItem(
                              value: 'edit',
                              child: Text(l10n.t('common.edit')),
                            ),
                            PopupMenuItem(
                              value: 'delete',
                              child: Text(l10n.t('common.delete')),
                            ),
                          ],
                        ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 12),
        ],
      ],
    );
  }
}
