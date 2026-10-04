import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

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
  final repo = TextEditingController();

  @override
  void initState() {
    super.initState();
    context.read<LoopFocus>().rememberProject(widget.id);
    _future = _load();
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

  Future<void> _reload() async {
    setState(() => _future = _load());
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder(
      future: _future,
      builder: (context, snap) {
        final l10n = context.watch<Appearance>();
        final theme = Theme.of(context);
        final title = snap.data?['title'] as String? ?? l10n.t('evidence.title');
        final updates = ((snap.data?['updates'] as List?) ?? const [])
            .map((e) => ProjectUpdate.fromJson(e as Map<String, dynamic>))
            .toList();
        return AppScaffold(
          body: !snap.hasData
              ? Center(child: snap.hasError ? Text(localizedError(snap.error!, l10n.t)) : const CircularProgressIndicator())
              : ListView(
                  padding: const EdgeInsets.fromLTRB(28, 64, 28, 140),
                  children: [
                    Text(l10n.t('evidence.title').toUpperCase(), style: theme.textTheme.displaySmall),
                    const SizedBox(height: 8),
                    Text(title, style: theme.textTheme.headlineSmall),
                    const SizedBox(height: 8),
                    Text(l10n.t('evidence.lead'), style: theme.textTheme.bodyLarge),
                    const SizedBox(height: 20),
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      children: [
                        FilledButton(onPressed: () => context.go('/projects/${widget.id}/roadmap'), child: Text(l10n.t('roadmap.title'))),
                        OutlinedButton(onPressed: () => _edit(snap.data!), child: Text(l10n.t('project.edit'))),
                        OutlinedButton(onPressed: () => _delete(snap.data!), child: Text(l10n.t('project.delete'))),
                      ],
                    ),
                    const SizedBox(height: 20),
                    FieldCard(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(l10n.t('github.title'), style: theme.textTheme.titleMedium),
                          const SizedBox(height: 12),
                          TextField(controller: repo, decoration: InputDecoration(labelText: l10n.t('github.placeholder'))),
                          const SizedBox(height: 12),
                          Wrap(
                            spacing: 8,
                            children: [
                              OutlinedButton(
                                onPressed: () async {
                                  await context.read<Session>().api.setRepo(widget.id, repo.text.trim());
                                  await _reload();
                                },
                                child: Text(l10n.t('github.save')),
                              ),
                              OutlinedButton(
                                onPressed: () async {
                                  final result = await context.read<Session>().api.syncGithub(widget.id);
                                  if (context.mounted) {
                                    final count = result['added'] ?? result['count'] ?? 0;
                                    ScaffoldMessenger.of(context).showSnackBar(
                                      SnackBar(content: Text(l10n.t('github.synced', {'count': count}))),
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
                    const SizedBox(height: 20),
                    Text(l10n.t('project.updates'), style: theme.textTheme.titleLarge),
                    const SizedBox(height: 12),
                    if (updates.isEmpty) Text(l10n.t('github.empty'), style: theme.textTheme.bodyLarge),
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
                              child: PopupMenuButton<String>(
                                onSelected: (value) async {
                                  final api = context.read<Session>().api;
                                  if (value == 'edit') {
                                    final content = TextEditingController(text: u.content);
                                    final ok = await showDialog<bool>(
                                      context: context,
                                      builder: (ctx) => AlertDialog(
                                        title: Text(l10n.t('common.edit')),
                                        content: TextField(controller: content),
                                        actions: [
                                          TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
                                          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('common.save'))),
                                        ],
                                      ),
                                    );
                                    if (ok == true && mounted) {
                                      await api.editUpdate(u.id, content.text.trim());
                                      await _reload();
                                    }
                                    content.dispose();
                                  } else if (value == 'delete') {
                                    await api.removeUpdate(u.id);
                                    await _reload();
                                  }
                                },
                                itemBuilder: (_) => [
                                  PopupMenuItem(value: 'edit', child: Text(l10n.t('common.edit'))),
                                  PopupMenuItem(value: 'delete', child: Text(l10n.t('common.delete'))),
                                ],
                              ),
                            ),
                          ],
                        ),
                      ),
                      const SizedBox(height: 12),
                    ],
                    FilledButton(onPressed: _addUpdate, child: Text(l10n.t('project.addUpdate'))),
                  ],
                ),
        );
      },
    );
  }

  Future<void> _edit(Map<String, dynamic> data) async {
    final l10n = context.read<Appearance>();
    final titleCtrl = TextEditingController(text: data['title'] as String? ?? '');
    final descCtrl = TextEditingController(text: data['description'] as String? ?? '');
    var status = (data['status'] as String?) ?? 'planning';
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setLocal) => AlertDialog(
          title: Text(l10n.t('project.edit')),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(controller: titleCtrl, decoration: InputDecoration(labelText: l10n.t('project.namePlaceholder'))),
              TextField(controller: descCtrl, decoration: InputDecoration(labelText: l10n.t('project.updatePlaceholder'))),
              DropdownButton<String>(
                value: status,
                items: const ['planning', 'active', 'paused', 'done', 'archived']
                    .map((s) => DropdownMenuItem(value: s, child: Text(s)))
                    .toList(),
                onChanged: (v) => setLocal(() => status = v ?? status),
              ),
            ],
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
            FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('common.save'))),
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
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('common.delete'))),
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
                items: const ['progress', 'blocker', 'note'].map((k) => DropdownMenuItem(value: k, child: Text(k))).toList(),
                onChanged: (v) => setLocal(() => kind = v ?? kind),
              ),
              TextField(controller: content, decoration: InputDecoration(labelText: l10n.t('project.updatePlaceholder'))),
            ],
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
            FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('project.post'))),
          ],
        ),
      ),
    );
    if (ok == true && content.text.trim().isNotEmpty && mounted) {
      await context.read<Session>().api.addUpdate(widget.id, kind, content.text.trim());
      await _reload();
    }
    content.dispose();
  }
}
