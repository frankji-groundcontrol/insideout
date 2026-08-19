import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';

class SettingsPage extends StatefulWidget {
  const SettingsPage({super.key, required this.workspaceId});

  final String workspaceId;

  @override
  State<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends State<SettingsPage> {
  late Future<({Workspace ws, List<WorkspaceMember> members})> _future;
  final title = TextEditingController();
  final description = TextEditingController();

  @override
  void initState() {
    super.initState();
    _future = _load();
  }

  @override
  void dispose() {
    title.dispose();
    description.dispose();
    super.dispose();
  }

  Future<({Workspace ws, List<WorkspaceMember> members})> _load() async {
    final api = context.read<Session>().api;
    final ws = await api.getWorkspace(widget.workspaceId);
    title.text = ws.title;
    description.text = ws.description;
    return (ws: ws, members: await api.listMembers(widget.workspaceId));
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
        return AppScaffold(
          title: l10n.t('workspace.settingsTitle'),
          body: !snap.hasData
              ? Center(child: snap.hasError ? Text('${snap.error}') : const CircularProgressIndicator())
              : ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    TextField(controller: title, decoration: InputDecoration(labelText: l10n.t('workspace.createNamePlaceholder'))),
                    const SizedBox(height: 8),
                    TextField(controller: description, decoration: InputDecoration(labelText: l10n.t('workspace.createDescPlaceholder'))),
                    const SizedBox(height: 8),
                    FilledButton(
                      onPressed: () async {
                        await context.read<Session>().api.updateWorkspace(
                              widget.workspaceId,
                              title: title.text.trim(),
                              description: description.text,
                            );
                        await _reload();
                      },
                      child: Text(l10n.t('common.save')),
                    ),
                    TextButton(
                      onPressed: () async {
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
                          await context.read<Session>().api.deleteWorkspace(widget.workspaceId);
                          if (context.mounted) context.go('/dashboard');
                        }
                      },
                      child: Text(l10n.t('workspace.deleteWorkspace')),
                    ),
                    ListTile(
                      title: Text(l10n.t('workspace.inviteCode', {'code': snap.data!.ws.code})),
                      subtitle: Text(l10n.t('workspace.inviteHint')),
                      trailing: IconButton(
                        icon: const Icon(Icons.copy),
                        onPressed: () => Clipboard.setData(ClipboardData(text: snap.data!.ws.code)),
                      ),
                    ),
                    const Divider(),
                    ...snap.data!.members.map(
                      (m) => ListTile(
                        title: Text(m.username),
                        subtitle: Text('${m.email} · ${l10n.t('workspace.role.${m.role}')}'),
                        trailing: PopupMenuButton<String>(
                          onSelected: (value) async {
                            final api = context.read<Session>().api;
                            if (value == 'remove') {
                              await api.removeMember(widget.workspaceId, m.userId);
                            } else {
                              await api.updateMemberRole(widget.workspaceId, m.userId, value);
                            }
                            await _reload();
                          },
                          itemBuilder: (_) => [
                            PopupMenuItem(value: 'admin', child: Text(l10n.t('workspace.role.admin'))),
                            PopupMenuItem(value: 'member', child: Text(l10n.t('workspace.role.member'))),
                            PopupMenuItem(value: 'remove', child: Text(l10n.t('workspace.removeMember'))),
                          ],
                        ),
                      ),
                    ),
                  ],
                ),
        );
      },
    );
  }
}
