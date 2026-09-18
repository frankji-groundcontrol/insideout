import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/loop_focus.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/field_card.dart';

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
    context.read<LoopFocus>().rememberWorkspace(widget.id);
    _future = _load();
  }

  Future<({Workspace ws, List<Project> projects})> _load() async {
    final api = context.read<Session>().api;
    final ws = await api.getWorkspace(widget.id);
    final projects = await api.listProjects(widget.id);
    final focus = context.read<LoopFocus>();
    if (focus.projectId == null && projects.isNotEmpty) {
      focus.rememberProject(projects.first.id, workspaceId: widget.id);
    }
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
        final theme = Theme.of(context);
        final title = snap.data?.ws.title ?? l10n.t('workspace.title');
        return AppScaffold(
          title: title,
          paintTitle: true,
          fab: FloatingActionButton(onPressed: _createProject, child: const Icon(Icons.add)),
          body: !snap.hasData
              ? Center(child: snap.hasError ? Text('${snap.error}') : const CircularProgressIndicator())
              : ListView(
                  padding: const EdgeInsets.fromLTRB(24, 8, 24, 140),
                  children: [
                    FieldCard(
                      child: Row(
                        children: [
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(l10n.t('workspace.inviteCode', {'code': snap.data!.ws.code}), style: theme.textTheme.titleMedium),
                                Text(l10n.t('workspace.inviteHint'), style: theme.textTheme.bodySmall),
                              ],
                            ),
                          ),
                          IconButton(
                            icon: const Icon(Icons.copy),
                            onPressed: () => Clipboard.setData(ClipboardData(text: snap.data!.ws.code)),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(height: 12),
                    FieldCard(
                      onTap: () => context.go('/workspace/${widget.id}/ideas'),
                      child: Text(l10n.t('workspace.ideas'), style: theme.textTheme.titleMedium),
                    ),
                    const SizedBox(height: 12),
                    FieldCard(
                      onTap: () => context.go('/workspace/${widget.id}/settings'),
                      child: Text(l10n.t('workspace.settings'), style: theme.textTheme.titleMedium),
                    ),
                    const SizedBox(height: 12),
                    FieldCard(onTap: () => _edit(snap.data!.ws), child: Text(l10n.t('common.edit'), style: theme.textTheme.titleMedium)),
                    const SizedBox(height: 12),
                    FieldCard(onTap: _delete, child: Text(l10n.t('workspace.deleteWorkspace'), style: theme.textTheme.titleMedium)),
                    const SizedBox(height: 24),
                    if (snap.data!.projects.isEmpty)
                      Text(l10n.t('project.empty'), style: theme.textTheme.bodyLarge)
                    else
                      for (final p in snap.data!.projects) ...[
                        FieldCard(
                          onTap: () {
                            context.read<LoopFocus>().rememberProject(p.id, workspaceId: widget.id);
                            context.go('/projects/${p.id}');
                          },
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(p.title, style: theme.textTheme.titleLarge),
                              Text(l10n.t('project.status.${p.status}'), style: theme.textTheme.bodySmall),
                            ],
                          ),
                        ),
                        const SizedBox(height: 12),
                      ],
                  ],
                ),
        );
      },
    );
  }

  Future<void> _edit(Workspace ws) async {
    final title = TextEditingController(text: ws.title);
    final desc = TextEditingController(text: ws.description);
    final l10n = context.read<Appearance>();
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
  }

  Future<void> _delete() async {
    final l10n = context.read<Appearance>();
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
  }
}
