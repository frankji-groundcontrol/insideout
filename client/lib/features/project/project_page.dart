import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/client.dart';
import '../../api/errors_l10n.dart';
import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/loop_focus.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/cobalt_chip.dart';
import '../../theme/field_card.dart';

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

  @override
  void initState() {
    super.initState();
    context.read<LoopFocus>().rememberProject(widget.id);
    _future = _load();
    _leafEvidence = _loadLeafEvidence();
  }

  Future<Map<String, dynamic>> _load() async {
    final api = context.read<Session>().api;
    final data = await api.getProject(widget.id);
    if (!mounted) return data;
    final ws = data['workspaceId'] as String?;
    context.read<LoopFocus>().rememberProject(widget.id, workspaceId: ws);
    repo.text = (data['repoUrl'] as String?) ?? '';
    return data;
  }

  @override
  void dispose() {
    repo.dispose();
    super.dispose();
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
      return Future.wait(
        done.map((n) async {
          final rows = await api.listNodeEvidence(n.id);
          return (title: n.title, rows: rows);
        }),
      );
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
    final l10n = context.read<Appearance>();
    final api = context.read<Session>().api;
    try {
      await api.decideProposal(update.id, accept: accept);
      if (accept) await _applyAcceptedItems(api, update);
    } on Object catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(localizedError(e, l10n.t))));
      }
      return;
    }
    if (mounted) await _reload();
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
        final title =
            snap.data?['title'] as String? ?? l10n.t('evidence.title');
        final updates = ((snap.data?['updates'] as List?) ?? const [])
            .map((e) => ProjectUpdate.fromJson(e as Map<String, dynamic>))
            .toList();
        return AppScaffold(
          body: !snap.hasData
              ? Center(
                  child: snap.hasError
                      ? Text(localizedError(snap.error!, l10n.t))
                      : const CircularProgressIndicator(),
                )
              : ListView(
                  padding: const EdgeInsets.fromLTRB(28, 64, 28, 140),
                  children: [
                    Text(
                      l10n.t('evidence.title').toUpperCase(),
                      style: theme.textTheme.displaySmall,
                    ),
                    const SizedBox(height: 8),
                    Text(title, style: theme.textTheme.headlineSmall),
                    const SizedBox(height: 8),
                    Text(
                      l10n.t('evidence.lead'),
                      style: theme.textTheme.bodyLarge,
                    ),
                    const SizedBox(height: 20),
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      children: [
                        FilledButton(
                          onPressed: () =>
                              context.go('/projects/${widget.id}/roadmap'),
                          child: Text(l10n.t('roadmap.title')),
                        ),
                        OutlinedButton(
                          onPressed: () => _edit(snap.data!),
                          child: Text(l10n.t('project.edit')),
                        ),
                        OutlinedButton(
                          onPressed: () => _delete(snap.data!),
                          child: Text(l10n.t('project.delete')),
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
                          const SizedBox(height: 12),
                          TextField(
                            controller: repo,
                            decoration: InputDecoration(
                              labelText: l10n.t('github.placeholder'),
                            ),
                          ),
                          const SizedBox(height: 12),
                          Wrap(
                            spacing: 8,
                            children: [
                              OutlinedButton(
                                onPressed: () async {
                                  await context.read<Session>().api.setRepo(
                                    widget.id,
                                    repo.text.trim(),
                                  );
                                  await _reload();
                                },
                                child: Text(l10n.t('github.save')),
                              ),
                              OutlinedButton(
                                onPressed: () async {
                                  final result = await context
                                      .read<Session>()
                                      .api
                                      .syncGithub(widget.id);
                                  if (context.mounted) {
                                    final count =
                                        result['added'] ?? result['count'] ?? 0;
                                    ScaffoldMessenger.of(context).showSnackBar(
                                      SnackBar(
                                        content: Text(
                                          l10n.t('github.synced', {
                                            'count': count,
                                          }),
                                        ),
                                      ),
                                    );
                                  }
                                },
                                child: Text(l10n.t('github.sync')),
                              ),
                            ],
                          ),
                        ],
                      ),
                    ),
                    FutureBuilder<
                      List<({String title, List<Map<String, dynamic>> rows})>
                    >(
                      future: _leafEvidence,
                      builder: (context, esnap) {
                        if (!esnap.hasData || esnap.data!.isEmpty)
                          return const SizedBox.shrink();
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
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Text(
                                      leaf.title,
                                      style: theme.textTheme.titleMedium,
                                    ),
                                    if (leaf.rows.isEmpty)
                                      Padding(
                                        padding: const EdgeInsets.only(top: 4),
                                        child: Text(
                                          l10n.t('evidence.noRows'),
                                          style: theme.textTheme.bodySmall,
                                        ),
                                      ),
                                    for (final row in leaf.rows)
                                      Padding(
                                        padding: const EdgeInsets.only(top: 4),
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
                            ShadeChip(label: u.kind, tone: ShadeTone.neutral),
                            const SizedBox(height: 8),
                            Text(u.content, style: theme.textTheme.titleMedium),
                            Text(u.createdAt, style: theme.textTheme.bodySmall),
                            Align(
                              alignment: Alignment.centerRight,
                              // FEAT-03: an agent proposal is decided, not
                              // edited — Accept/Reject own the record.
                              child: u.kind == 'agent_proposal'
                                  ? Wrap(
                                      spacing: 8,
                                      children: [
                                        FilledButton(
                                          onPressed: () =>
                                              _decide(u, accept: true),
                                          child: Text(
                                            l10n.t('proposal.accept'),
                                          ),
                                        ),
                                        OutlinedButton(
                                          onPressed: () =>
                                              _decide(u, accept: false),
                                          child: Text(
                                            l10n.t('proposal.reject'),
                                          ),
                                        ),
                                      ],
                                    )
                                  : PopupMenuButton<String>(
                                      onSelected: (value) async {
                                        final api = context.read<Session>().api;
                                        if (value == 'edit') {
                                          final content = TextEditingController(
                                            text: u.content,
                                          );
                                          final ok = await showDialog<bool>(
                                            context: context,
                                            builder: (ctx) => AlertDialog(
                                              title: Text(
                                                l10n.t('common.edit'),
                                              ),
                                              content: TextField(
                                                controller: content,
                                              ),
                                              actions: [
                                                TextButton(
                                                  onPressed: () =>
                                                      Navigator.pop(ctx, false),
                                                  child: Text(
                                                    l10n.t('common.cancel'),
                                                  ),
                                                ),
                                                FilledButton(
                                                  onPressed: () =>
                                                      Navigator.pop(ctx, true),
                                                  child: Text(
                                                    l10n.t('common.save'),
                                                  ),
                                                ),
                                              ],
                                            ),
                                          );
                                          if (ok == true && mounted) {
                                            await api.editUpdate(
                                              u.id,
                                              content.text.trim(),
                                            );
                                            await _reload();
                                          }
                                          content.dispose();
                                        } else if (value == 'delete') {
                                          await api.removeUpdate(u.id);
                                          await _reload();
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
                    FilledButton(
                      onPressed: _addUpdate,
                      child: Text(l10n.t('project.addUpdate')),
                    ),
                  ],
                ),
        );
      },
    );
  }

  Future<void> _edit(Map<String, dynamic> data) async {
    final l10n = context.read<Appearance>();
    final titleCtrl = TextEditingController(
      text: data['title'] as String? ?? '',
    );
    final descCtrl = TextEditingController(
      text: data['description'] as String? ?? '',
    );
    var status = (data['status'] as String?) ?? 'planning';
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setLocal) => AlertDialog(
          title: Text(l10n.t('project.edit')),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: titleCtrl,
                decoration: InputDecoration(
                  labelText: l10n.t('project.namePlaceholder'),
                ),
              ),
              TextField(
                controller: descCtrl,
                decoration: InputDecoration(
                  labelText: l10n.t('project.updatePlaceholder'),
                ),
              ),
              DropdownButton<String>(
                value: status,
                items:
                    const ['planning', 'active', 'paused', 'done', 'archived']
                        .map((s) => DropdownMenuItem(value: s, child: Text(s)))
                        .toList(),
                onChanged: (v) => setLocal(() => status = v ?? status),
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
              child: Text(l10n.t('common.save')),
            ),
          ],
        ),
      ),
    );
    if (ok == true && mounted) {
      await context.read<Session>().api.updateProject(
        widget.id,
        title: titleCtrl.text.trim(),
        description: descCtrl.text,
        status: status,
        ownerId: data['ownerId'] as String?,
      );
      await _reload();
    }
    titleCtrl.dispose();
    descCtrl.dispose();
  }

  Future<void> _delete(Map<String, dynamic> data) async {
    final l10n = context.read<Appearance>();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.t('project.deleteConfirm')),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: Text(l10n.t('common.cancel')),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(l10n.t('common.delete')),
          ),
        ],
      ),
    );
    if (ok == true && mounted) {
      final wsId = data['workspaceId'] as String;
      await context.read<Session>().api.deleteProject(widget.id);
      if (context.mounted) context.go('/workspace/$wsId');
    }
  }

  Future<void> _addUpdate() async {
    final l10n = context.read<Appearance>();
    final content = TextEditingController();
    var kind = 'note';
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setLocal) => AlertDialog(
          title: Text(l10n.t('project.addUpdate')),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              DropdownButton<String>(
                value: kind,
                items: const ['progress', 'blocker', 'note']
                    .map((k) => DropdownMenuItem(value: k, child: Text(k)))
                    .toList(),
                onChanged: (v) => setLocal(() => kind = v ?? kind),
              ),
              TextField(
                controller: content,
                decoration: InputDecoration(
                  labelText: l10n.t('project.updatePlaceholder'),
                ),
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
              child: Text(l10n.t('project.post')),
            ),
          ],
        ),
      ),
    );
    if (ok == true && content.text.trim().isNotEmpty && mounted) {
      await context.read<Session>().api.addUpdate(
        widget.id,
        kind,
        content.text.trim(),
      );
      await _reload();
    }
    content.dispose();
  }
}
