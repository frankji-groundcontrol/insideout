import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';

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
    _future = context.read<Session>().api.getProject(widget.id);
  }

  @override
  void dispose() {
    repo.dispose();
    super.dispose();
  }

  Future<void> _reload() async {
    setState(() => _future = context.read<Session>().api.getProject(widget.id));
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder(
      future: _future,
      builder: (context, snap) {
        final l10n = context.watch<Appearance>();
        final title = snap.data?['title'] as String? ?? l10n.t('project.title');
        final updates = ((snap.data?['updates'] as List?) ?? const [])
            .map((e) => ProjectUpdate.fromJson(e as Map<String, dynamic>))
            .toList();
        if (snap.hasData) repo.text = (snap.data!['repoUrl'] as String?) ?? '';
        return AppScaffold(
          title: title,
          body: !snap.hasData
              ? Center(child: snap.hasError ? Text('${snap.error}') : const CircularProgressIndicator())
              : ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    ListTile(
                      title: Text(l10n.t('roadmap.title')),
                      leading: const Icon(Icons.account_tree_outlined),
                      onTap: () => context.go('/projects/${widget.id}/roadmap'),
                    ),
                    ListTile(
                      title: Text(l10n.t('project.edit')),
                      leading: const Icon(Icons.edit_outlined),
                      onTap: () async {
                        final data = snap.data!;
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
                      },
                    ),
                    ListTile(
                      title: Text(l10n.t('project.delete')),
                      leading: const Icon(Icons.delete_outline),
                      onTap: () async {
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
                          final wsId = snap.data!['workspaceId'] as String;
                          await context.read<Session>().api.deleteProject(widget.id);
                          if (context.mounted) context.go('/workspace/$wsId');
                        }
                      },
                    ),
                    TextField(controller: repo, decoration: InputDecoration(labelText: l10n.t('github.placeholder'))),
                    const SizedBox(height: 8),
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
                    const Divider(height: 32),
                    Text(l10n.t('project.updates'), style: Theme.of(context).textTheme.titleMedium),
                    ...updates.map(
                      (u) => ListTile(
                        title: Text(u.content),
                        subtitle: Text('${u.kind} · ${u.createdAt}'),
                        trailing: PopupMenuButton<String>(
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
                    ),
                    const SizedBox(height: 12),
                    FilledButton(
                      onPressed: () async {
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
                      },
                      child: Text(l10n.t('project.addUpdate')),
                    ),
                  ],
                ),
        );
      },
    );
  }
}
