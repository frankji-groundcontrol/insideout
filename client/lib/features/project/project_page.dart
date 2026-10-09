import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/client.dart';
import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/loop_focus.dart';
import '../../chrome/product_dock.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_field.dart';
import '../../theme/block_form.dart';
import '../../theme/block_modal.dart';
import '../../theme/block_state.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';
import '../../theme/field_card.dart';
import '../../widgets/guarded_action.dart';

/// Clears the top-right utility block on the field; the body extends behind
/// the dock (AppScaffold extendBody).
const double _topInset = 64;
const double _bottomInset = ProductDock.height + Cobalt.space10 + Cobalt.space4;

class ProjectPage extends StatefulWidget {
  const ProjectPage({super.key, required this.id});

  final String id;

  @override
  State<ProjectPage> createState() => _ProjectPageState();
}

class _ProjectPageState extends State<ProjectPage> {
  late Future<Map<String, dynamic>> _future;
  late Future<List<({String title, List<Map<String, dynamic>> rows})>>
  _leafEvidence;
  final repo = TextEditingController();
  final _scroll = ScrollController();
  Timer? _flashTimer;
  var _flash = false;
  var _lastPulse = 0;
  late final LoopFocus _focus;

  @override
  void initState() {
    super.initState();
    _focus = context.read<LoopFocus>();
    _focus.rememberProject(widget.id);
    // The dock bumps evidencePulse when its Evidence step is tapped while
    // this page is already the destination; one-shot per bump (the counter
    // also filters the focus's other notifications).
    _lastPulse = _focus.evidencePulse;
    _focus.addListener(_onEvidencePulse);
    _future = _load();
    _leafEvidence = _loadLeafEvidence();
  }

  @override
  void dispose() {
    // Already deactivated: context.read would throw and skip the timer cancel.
    _flashTimer?.cancel();
    _focus.removeListener(_onEvidencePulse);
    _scroll.dispose();
    repo.dispose();
    super.dispose();
  }

  /// Pulse from the dock's Evidence item: scroll the evidence header back
  /// into view and flash it, so the tap reads as "you are here".
  void _onEvidencePulse() {
    if (_focus.evidencePulse == _lastPulse) return;
    _lastPulse = _focus.evidencePulse;
    if (!mounted) return;
    setState(() => _flash = true);
    _flashTimer?.cancel();
    _flashTimer = Timer(const Duration(seconds: 1), () {
      if (mounted) setState(() => _flash = false);
    });
    if (_scroll.hasClients) {
      // A ticker animation does not move on the pump that shows the flash.
      _scroll.jumpTo(0);
    }
  }

  Future<Map<String, dynamic>> _load() async {
    final api = context.read<Session>().api;
    final data = await api.getProject(widget.id);
    if (!mounted) return data;
    final ws = data['workspaceId'] as String?;
    _focus.rememberProject(widget.id, workspaceId: ws);
    repo.text = (data['repoUrl'] as String?) ?? '';
    return data;
  }

  /// Evidence rows for the done leaves (cap 5), so the Evidence screen
  /// shows what finished work actually produced.
  Future<List<({String title, List<Map<String, dynamic>> rows})>>
  _loadLeafEvidence() async {
    final api = context.read<Session>().api;
    try {
      final nodes = await api.listRoadmap(widget.id);
      final done = nodes
          .where((n) => n.parentId != null && n.status == 'done')
          .take(5)
          .toList();
      final leaves = await Future.wait(
        done.map((n) async {
          final rows = await api.listNodeEvidence(n.id);
          return (title: n.title, rows: rows);
        }),
      );
      return leaves;
    } on Exception {
      return const [];
    }
  }

  Future<void> _reload() async {
    setState(() {
      _future = _load();
      _leafEvidence = _loadLeafEvidence();
    });
  }

  // FEAT-03: the proposal decision loop — the server resolves authority
  // (project owner or workspace admin); the web just surfaces the outcome.
  Future<void> _decide(ProjectUpdate update, {required bool accept}) async {
    final api = context.read<Session>().api;
    final done = await guardWrite(context, () async {
      await api.decideProposal(update.id, accept: accept);
      if (accept) await _applyAcceptedItems(api, update);
    });
    if (done && mounted) await _reload();
  }

  // Accepting a proposal changes the roadmap: each add_node item becomes a
  // node. parentHint matches an existing node's title; empty means a root.
  Future<void> _applyAcceptedItems(ApiClient api, ProjectUpdate update) async {
    final nodes = await api.listRoadmap(widget.id);
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
        widget.id,
        item.title.trim(),
        parentId: parentId,
      );
      nodes.add(created);
    }
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder(
      future: _future,
      builder: (context, snap) {
        final l10n = context.watch<Appearance>();
        final theme = Theme.of(context);
        final palette = Cobalt.of(context);
        final title =
            snap.data?['title'] as String? ?? l10n.t('evidence.title');
        final updates = ((snap.data?['updates'] as List?) ?? const [])
            .map((e) => ProjectUpdate.fromJson(e as Map<String, dynamic>))
            .toList();
        return AppScaffold(
          body: !snap.hasData
              ? snap.hasError
                    ? BlockError(error: snap.error, onRetry: _reload)
                    : const BlockLoading()
              : Center(
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(
                      maxWidth: Cobalt.contentMaxWidth,
                    ),
                    child: ListView(
                      controller: _scroll,
                      padding: const EdgeInsets.fromLTRB(
                        28,
                        _topInset,
                        28,
                        _bottomInset,
                      ),
                      children: [
                        // The dock's Evidence pulse flashes this header — a
                        // brandSubtle fill behind a strokeStrong border that
                        // fades back out (Cobalt-Neo tokens, 0 radius).
                        AnimatedContainer(
                          key: const ValueKey('evidence-pulse-flash'),
                          duration: const Duration(milliseconds: 180),
                          padding: const EdgeInsets.all(12),
                          decoration: BoxDecoration(
                            color: _flash
                                ? palette.brandSubtle
                                : palette.brandSubtle.withValues(alpha: 0),
                            border: Border.all(
                              color: _flash
                                  ? palette.strokeStrong
                                  : palette.strokeStrong.withValues(alpha: 0),
                              width: Cobalt.borderContainer,
                            ),
                          ),
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Text(
                                l10n.t('evidence.title').toUpperCase(),
                                style: theme.textTheme.labelSmall,
                              ),
                              const SizedBox(height: 8),
                              Text(title, style: theme.textTheme.displaySmall),
                              const SizedBox(height: 8),
                              Text(
                                l10n.t('evidence.lead'),
                                style: theme.textTheme.bodyLarge,
                              ),
                            ],
                          ),
                        ),
                        const SizedBox(height: 20),
                        Wrap(
                          spacing: Cobalt.space2,
                          runSpacing: Cobalt.space2,
                          children: [
                            BlockButton(
                              label: l10n.t('roadmap.title'),
                              icon: Icons.account_tree_outlined,
                              onPressed: () =>
                                  context.go('/projects/${widget.id}/roadmap'),
                            ),
                            BlockButton(
                              label: l10n.t('project.edit'),
                              icon: Icons.edit_outlined,
                              variant: BlockButtonVariant.secondary,
                              onPressed: () => _edit(snap.data!),
                            ),
                            BlockButton(
                              label: l10n.t('project.delete'),
                              icon: Icons.delete_outline,
                              variant: BlockButtonVariant.danger,
                              onPressed: () => _delete(snap.data!),
                            ),
                          ],
                        ),
                        const SizedBox(height: 20),
                        FieldCard(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                l10n.t('github.title'),
                                style: theme.textTheme.titleMedium,
                              ),
                              const SizedBox(height: Cobalt.space3),
                              BlockField(
                                controller: repo,
                                label: l10n.t('github.placeholder'),
                              ),
                              const SizedBox(height: Cobalt.space3),
                              Wrap(
                                spacing: Cobalt.space2,
                                runSpacing: Cobalt.space2,
                                children: [
                                  BlockButton(
                                    label: l10n.t('github.save'),
                                    variant: BlockButtonVariant.secondary,
                                    onPressed: () async {
                                      final done = await guardWrite(
                                        context,
                                        () =>
                                            context.read<Session>().api.setRepo(
                                              widget.id,
                                              repo.text.trim(),
                                            ),
                                      );
                                      if (done && mounted) await _reload();
                                    },
                                  ),
                                  BlockButton(
                                    label: l10n.t('github.sync'),
                                    icon: Icons.sync,
                                    variant: BlockButtonVariant.secondary,
                                    onPressed: () async {
                                      await guardWrite(context, () async {
                                        final result = await context
                                            .read<Session>()
                                            .api
                                            .syncGithub(widget.id);
                                        if (context.mounted) {
                                          final count =
                                              result['added'] ??
                                              result['count'] ??
                                              0;
                                          ScaffoldMessenger.of(
                                            context,
                                          ).showSnackBar(
                                            SnackBar(
                                              content: Text(
                                                l10n.t('github.synced', {
                                                  'count': count,
                                                }),
                                              ),
                                            ),
                                          );
                                        }
                                      });
                                    },
                                  ),
                                ],
                              ),
                            ],
                          ),
                        ),
                        FutureBuilder<
                          List<
                            ({String title, List<Map<String, dynamic>> rows})
                          >
                        >(
                          future: _leafEvidence,
                          builder: (context, esnap) {
                            if (!esnap.hasData || esnap.data!.isEmpty) {
                              return const SizedBox.shrink();
                            }
                            return Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(
                                  l10n.t('evidence.doneTitle'),
                                  style: theme.textTheme.titleLarge,
                                ),
                                const SizedBox(height: 12),
                                for (final leaf in esnap.data!)
                                  FieldCard(
                                    child: Column(
                                      crossAxisAlignment:
                                          CrossAxisAlignment.start,
                                      children: [
                                        Text(
                                          leaf.title,
                                          style: theme.textTheme.titleMedium,
                                        ),
                                        if (leaf.rows.isEmpty)
                                          Padding(
                                            padding: const EdgeInsets.only(
                                              top: 4,
                                            ),
                                            child: Text(
                                              l10n.t('evidence.noRows'),
                                              style: theme.textTheme.bodySmall,
                                            ),
                                          ),
                                        for (final row in leaf.rows)
                                          Padding(
                                            padding: const EdgeInsets.only(
                                              top: 4,
                                            ),
                                            child: Text(
                                              '${row['kind'] ?? ''}: ${row['detail'] ?? ''}',
                                              style: theme.textTheme.bodySmall,
                                              maxLines: 2,
                                              overflow: TextOverflow.ellipsis,
                                            ),
                                          ),
                                      ],
                                    ),
                                  ),
                                const SizedBox(height: 20),
                              ],
                            );
                          },
                        ),
                        const SizedBox(height: 20),
                        Text(
                          l10n.t('project.updates'),
                          style: theme.textTheme.titleLarge,
                        ),
                        const SizedBox(height: 12),
                        if (updates.isEmpty)
                          Text(
                            l10n.t('github.empty'),
                            style: theme.textTheme.bodyLarge,
                          ),
                        for (final u in updates) ...[
                          FieldCard(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                ShadeChip(
                                  label: u.kind,
                                  tone: ShadeTone.neutral,
                                ),
                                const SizedBox(height: 8),
                                Text(
                                  u.content,
                                  style: theme.textTheme.titleMedium,
                                ),
                                Text(
                                  u.createdAt,
                                  style: theme.textTheme.bodySmall,
                                ),
                                Align(
                                  alignment: Alignment.centerRight,
                                  // FEAT-03: an agent proposal is decided, not
                                  // edited — Accept/Reject own the record.
                                  child:
                                      u.kind == 'agent_proposal' &&
                                          u.decision.isEmpty
                                      ? Wrap(
                                          spacing: Cobalt.space2,
                                          children: [
                                            BlockButton(
                                              label: l10n.t('proposal.accept'),
                                              size: BlockButtonSize.sm,
                                              onPressed: () =>
                                                  _decide(u, accept: true),
                                            ),
                                            BlockButton(
                                              label: l10n.t('proposal.reject'),
                                              size: BlockButtonSize.sm,
                                              variant:
                                                  BlockButtonVariant.secondary,
                                              onPressed: () =>
                                                  _decide(u, accept: false),
                                            ),
                                          ],
                                        )
                                      : PopupMenuButton<String>(
                                          onSelected: (value) async {
                                            final api = context
                                                .read<Session>()
                                                .api;
                                            if (value == 'edit') {
                                              // The form state owns its controller
                                              // and disposes it on unmount — no
                                              // controller outlives the pop.
                                              final values = await showBlockForm(
                                                context: context,
                                                title: l10n.t('common.edit'),
                                                confirmLabel: l10n.t(
                                                  'common.save',
                                                ),
                                                fields: [
                                                  BlockFormField(
                                                    label: l10n.t(
                                                      'project.updatePlaceholder',
                                                    ),
                                                    initial: u.content,
                                                  ),
                                                ],
                                              );
                                              if (values != null &&
                                                  context.mounted) {
                                                final done = await guardWrite(
                                                  context,
                                                  () => api.editUpdate(
                                                    u.id,
                                                    values.first.trim(),
                                                  ),
                                                );
                                                if (done && context.mounted) {
                                                  await _reload();
                                                }
                                              }
                                            } else if (value == 'delete') {
                                              final done = await guardWrite(
                                                context,
                                                () => api.removeUpdate(u.id),
                                              );
                                              if (done && context.mounted) {
                                                await _reload();
                                              }
                                            }
                                          },
                                          itemBuilder: (_) => [
                                            PopupMenuItem(
                                              value: 'edit',
                                              child: Text(
                                                l10n.t('common.edit'),
                                              ),
                                            ),
                                            PopupMenuItem(
                                              value: 'delete',
                                              child: Text(
                                                l10n.t('common.delete'),
                                              ),
                                            ),
                                          ],
                                        ),
                                ),
                              ],
                            ),
                          ),
                          const SizedBox(height: 12),
                        ],
                        BlockButton(
                          label: l10n.t('project.addUpdate'),
                          icon: Icons.add,
                          expand: true,
                          onPressed: _addUpdate,
                        ),
                      ],
                    ),
                  ),
                ),
        );
      },
    );
  }

  Future<void> _edit(Map<String, dynamic> data) async {
    final t = context.read<Appearance>().t;
    final navigator = Navigator.of(context);
    final titleCtrl = TextEditingController(
      text: data['title'] as String? ?? '',
    );
    final descCtrl = TextEditingController(
      text: data['description'] as String? ?? '',
    );
    var status = (data['status'] as String?) ?? 'planning';
    final ok = await showBlockModal<bool>(
      context: context,
      title: t('project.edit'),
      body: StatefulBuilder(
        builder: (ctx, setLocal) => Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            BlockField(
              controller: titleCtrl,
              label: t('project.namePlaceholder'),
            ),
            const SizedBox(height: Cobalt.space4),
            BlockField(
              controller: descCtrl,
              label: t('project.updatePlaceholder'),
            ),
            const SizedBox(height: Cobalt.space4),
            BlockSelect<String>(
              label: t('project.statusLabel'),
              value: status,
              options: [
                for (final s in const [
                  'planning',
                  'active',
                  'paused',
                  'done',
                  'archived',
                ])
                  (value: s, label: t('project.status.$s')),
              ],
              onChanged: (v) => setLocal(() => status = v ?? status),
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
          label: t('common.save'),
          onPressed: () => navigator.pop(true),
        ),
      ],
    );
    if (ok == true && mounted) {
      final done = await guardWrite(
        context,
        () => context.read<Session>().api.updateProject(
          widget.id,
          title: titleCtrl.text.trim(),
          description: descCtrl.text,
          status: status,
          ownerId: data['ownerId'] as String?,
        ),
      );
      if (done && mounted) await _reload();
    }
    // Not disposed: the pop animation keeps the dialog's TextFields mounted
    // past this await, and the guard's SnackBar rebuilds the overlay — a
    // disposed controller would throw. GC'd once the dialog unmounts.
  }

  Future<void> _delete(Map<String, dynamic> data) async {
    final t = context.read<Appearance>().t;
    final ok = await showBlockConfirm(
      context: context,
      title: t('project.delete'),
      message: t('project.deleteConfirm'),
      confirmText: t('common.delete'),
      danger: true,
    );
    if (ok == true && mounted) {
      final wsId = data['workspaceId'] as String;
      final done = await guardWrite(
        context,
        () => context.read<Session>().api.deleteProject(widget.id),
      );
      if (done && mounted) context.go('/workspace/$wsId');
    }
  }

  Future<void> _addUpdate() async {
    final t = context.read<Appearance>().t;
    final navigator = Navigator.of(context);
    final content = TextEditingController();
    var kind = 'note';
    final ok = await showBlockModal<bool>(
      context: context,
      title: t('project.addUpdate'),
      body: StatefulBuilder(
        builder: (ctx, setLocal) => Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            BlockSelect<String>(
              value: kind,
              options: [
                for (final k in const ['progress', 'blocker', 'note'])
                  (value: k, label: t('project.updateKind.$k')),
              ],
              onChanged: (v) => setLocal(() => kind = v ?? kind),
            ),
            const SizedBox(height: Cobalt.space4),
            BlockField(
              controller: content,
              label: t('project.updatePlaceholder'),
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
          label: t('project.post'),
          onPressed: () => navigator.pop(true),
        ),
      ],
    );
    if (ok == true && content.text.trim().isNotEmpty && mounted) {
      final done = await guardWrite(
        context,
        () => context.read<Session>().api.addUpdate(
          widget.id,
          kind,
          content.text.trim(),
        ),
      );
      if (done && mounted) await _reload();
    }
    // Not disposed: the pop animation keeps the dialog's TextField mounted
    // past this await, and the guard's SnackBar rebuilds the overlay — a
    // disposed controller would throw. GC'd once the dialog unmounts.
  }
}
