import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';

class WorkspacePage extends StatefulWidget {
  const WorkspacePage({super.key, required this.id});

  final String id;

  @override
  State<WorkspacePage> createState() => _WorkspacePageState();
}

class _WorkspacePageState extends State<WorkspacePage> {
  late Future<({Workspace ws, List<Project> projects})> _future;

  @override
  void initState() {
    super.initState();
    _future = _load();
  }

  Future<({Workspace ws, List<Project> projects})> _load() async {
    final api = context.read<Session>().api;
    final ws = await api.getWorkspace(widget.id);
    final projects = await api.listProjects(widget.id);
    return (ws: ws, projects: projects);
  }

  Future<void> _createProject() async {
    final l10n = context.read<Appearance>();
    final title = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.t('project.newProject')),
        content: TextField(controller: title, decoration: InputDecoration(labelText: l10n.t('project.namePlaceholder'))),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('project.create'))),
        ],
      ),
    );
    if (ok == true && title.text.trim().isNotEmpty && mounted) {
      await context.read<Session>().api.createProject(widget.id, title.text.trim(), '');
      setState(() => _future = _load());
    }
    title.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder(
      future: _future,
      builder: (context, snap) {
        final l10n = context.watch<Appearance>();
        final title = snap.data?.ws.title ?? l10n.t('workspace.title');
        return AppScaffold(
          title: title,
          fab: FloatingActionButton(onPressed: _createProject, child: const Icon(Icons.add)),
          body: !snap.hasData
              ? Center(child: snap.hasError ? Text('${snap.error}') : const CircularProgressIndicator())
              : ListView(
                  children: [
                    ListTile(
                      title: Text(l10n.t('workspace.inviteCode', {'code': snap.data!.ws.code})),
                      subtitle: Text(l10n.t('workspace.inviteHint')),
                      trailing: IconButton(
                        icon: const Icon(Icons.copy),
                        onPressed: () => Clipboard.setData(ClipboardData(text: snap.data!.ws.code)),
                      ),
                    ),
                    ListTile(title: Text(l10n.t('workspace.ideas')), leading: const Icon(Icons.lightbulb_outline), onTap: () => context.go('/workspace/${widget.id}/ideas')),
                    ListTile(title: Text(l10n.t('workspace.settings')), leading: const Icon(Icons.settings_outlined), onTap: () => context.go('/workspace/${widget.id}/settings')),
                    ListTile(
                      title: Text(l10n.t('common.edit')),
                      leading: const Icon(Icons.edit_outlined),
                      onTap: () async {
                        final ws = snap.data!.ws;
                        final title = TextEditingController(text: ws.title);
                        final desc = TextEditingController(text: ws.description);
                        final ok = await showDialog<bool>(
                          context: context,
                          builder: (ctx) => AlertDialog(
                            title: Text(l10n.t('common.edit')),
                            content: Column(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                TextField(controller: title, decoration: InputDecoration(labelText: l10n.t('workspace.createNamePlaceholder'))),
                                TextField(controller: desc, decoration: InputDecoration(labelText: l10n.t('workspace.createDescPlaceholder'))),
                              ],
                            ),
                            actions: [
                              TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
                              FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('common.save'))),
                            ],
                          ),
                        );
                        if (ok == true && mounted) {
                          await context.read<Session>().api.updateWorkspace(widget.id, title: title.text.trim(), description: desc.text);
                          setState(() => _future = _load());
                        }
                        title.dispose();
                        desc.dispose();
                      },
                    ),
                    ListTile(
                      title: Text(l10n.t('workspace.deleteWorkspace')),
                      leading: const Icon(Icons.delete_outline),
                      onTap: () async {
                        final ok = await showDialog<bool>(
                          context: context,
                          builder: (ctx) => AlertDialog(
                            title: Text(l10n.t('workspace.deleteWorkspaceConfirm')),
                            actions: [
                              TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
                              FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('common.delete'))),
                            ],
                          ),
                        );
                        if (ok == true && mounted) {
                          await context.read<Session>().api.deleteWorkspace(widget.id);
                          if (context.mounted) context.go('/dashboard');
                        }
                      },
                    ),
                    const Divider(),
                    if (snap.data!.projects.isEmpty) ListTile(title: Text(l10n.t('project.empty'))),
                    ...snap.data!.projects.map(
                      (p) => ListTile(
                        title: Text(p.title),
                        subtitle: Text(l10n.t('project.status.${p.status}')),
                        onTap: () => context.go('/projects/${p.id}'),
                      ),
                    ),
                  ],
                ),
        );
      },
    );
  }
}
