import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/errors_l10n.dart';
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
class ProjectUpdatesFeed extends StatefulWidget {
  const ProjectUpdatesFeed({
    super.key,
    required this.projectId,
    required this.updates,
    required this.onChanged,
    required this.onEdit,
    required this.onRemove,
    required this.canDecide,
    this.nextCursor,
  });

  /// The project whose older pages "Load older updates" reads.
  final String projectId;
  final List<ProjectUpdate> updates;
  final Future<void> Function() onChanged;
  final Future<void> Function(ProjectUpdate update, String content) onEdit;
  final Future<void> Function(ProjectUpdate update) onRemove;

  /// The project owner or an admin (the view's canWritePlan): only they
  /// decide a proposal, so only they see Accept and Reject (FE-05).
  final bool canDecide;

  /// Where the older updates start; null when [updates] reach the first one.
  final String? nextCursor;

  @override
  State<ProjectUpdatesFeed> createState() => _ProjectUpdatesFeedState();
}

class _ProjectUpdatesFeedState extends State<ProjectUpdatesFeed> {
  /// Proposals whose decision is on the wire (id -> whether it is an accept)
  /// and proposals whose decision has landed. Both leave Accept and Reject
  /// off, the second even while the list this feed holds does not show the
  /// decision yet: every further decision adds another history note every
  /// member reads, and an opposite one flips a decision whose nodes already
  /// exist (rv4-FR-F02, F08, F15).
  final _deciding = <String, bool>{};
  final _decided = <String>{};

  /// Older pages read with "Load older updates" (rv5-PARB-07). The page
  /// hands a new list on every rebuild, so the history starts over only when
  /// the project or its newest update changes, or after this feed wrote to
  /// an older row (the reload does not carry that row's new state).
  var _older = <ProjectUpdate>[];
  late String? _cursor = widget.nextCursor;
  var _loading = false;
  var _olderStale = false;

  @override
  void didUpdateWidget(ProjectUpdatesFeed old) {
    super.didUpdateWidget(old);
    if (_olderStale ||
        old.projectId != widget.projectId ||
        old.updates.firstOrNull?.id != widget.updates.firstOrNull?.id) {
      _older = [];
      _cursor = widget.nextCursor;
      _olderStale = false;
    }
  }

  /// Called before a write's reload: a write to an older row resets the
  /// history on the next update.
  void _wrote(ProjectUpdate u) {
    if (_older.any((o) => o.id == u.id)) _olderStale = true;
  }

  Future<void> _loadOlder() async {
    final cursor = _cursor;
    if (cursor == null || _loading) return;
    final api = context.read<Session>().api;
    final t = context.read<Appearance>().t;
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _loading = true);
    try {
      final page = await api.listProjectUpdates(
        widget.projectId,
        before: cursor,
      );
      if (mounted) {
        setState(() {
          _older = [..._older, ...page.updates];
          _cursor = page.nextCursor;
        });
      }
    } catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(localizedError(e, t))));
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  // FEAT-03: the proposal decision loop — the server resolves authority
  // (project owner or workspace admin); the web just surfaces the outcome.
  // Accepting also applies the proposal's items: decideProposal asks for that
  // (apply defaults to true) and the server does it in the decision's own
  // transaction, once (rv4-FEAT-02). This client creates no node of its own —
  // a second pass here would double every node — and onChanged re-reads the
  // project and the roadmap the server just changed.
  Future<void> _decide(ProjectUpdate update, {required bool accept}) async {
    final id = update.id;
    // Checked here as well as by the disabled buttons: a second tap can land
    // before the frame that disables them.
    if (_deciding.containsKey(id) || _decided.contains(id)) return;
    final api = context.read<Session>().api;
    setState(() => _deciding[id] = accept);
    final done = await guardWrite(
      context,
      () => api.decideProposal(id, accept: accept),
    );
    if (!mounted) return;
    setState(() {
      _deciding.remove(id);
      if (done) _decided.add(id);
    });
    if (!done) return;
    _wrote(update);
    await widget.onChanged();
  }

  Widget _decision(ProjectUpdate u, Appearance l10n) {
    final pressed = _deciding[u.id];
    final off = pressed != null || _decided.contains(u.id);
    return Wrap(
      spacing: Cobalt.space2,
      children: [
        BlockButton(
          label: l10n.t('proposal.accept'),
          size: BlockButtonSize.sm,
          busy: pressed == true,
          onPressed: off ? null : () => _decide(u, accept: true),
        ),
        BlockButton(
          label: l10n.t('proposal.reject'),
          size: BlockButtonSize.sm,
          variant: BlockButtonVariant.secondary,
          busy: pressed == false,
          onPressed: off ? null : () => _decide(u, accept: false),
        ),
      ],
    );
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
        if (widget.updates.isEmpty)
          Text(l10n.t('github.empty'), style: theme.textTheme.bodyLarge),
        for (final u in [...widget.updates, ..._older]) ...[
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
                      ? (widget.canDecide
                            ? _decision(u, l10n)
                            : Text(
                                l10n.t('proposal.pending'),
                                style: theme.textTheme.bodySmall,
                              ))
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
                                    required: true,
                                  ),
                                ],
                              );
                              if (values != null && context.mounted) {
                                final done = await guardWrite(
                                  context,
                                  () => widget.onEdit(u, values.first.trim()),
                                );
                                if (done && context.mounted) {
                                  _wrote(u);
                                  await widget.onChanged();
                                }
                              }
                            } else if (value == 'delete') {
                              final done = await guardWrite(
                                context,
                                () => widget.onRemove(u),
                              );
                              if (done && context.mounted) {
                                _wrote(u);
                                await widget.onChanged();
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
        if (_cursor != null) ...[
          BlockButton(
            label: l10n.t('project.loadOlder'),
            variant: BlockButtonVariant.secondary,
            expand: true,
            busy: _loading,
            onPressed: _loadOlder,
          ),
          const SizedBox(height: 12),
        ],
      ],
    );
  }
}
