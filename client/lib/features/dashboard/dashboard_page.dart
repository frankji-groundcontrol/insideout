import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';

class DashboardPage extends StatefulWidget {
  const DashboardPage({super.key});

  @override
  State<DashboardPage> createState() => _DashboardPageState();
}

class _DashboardPageState extends State<DashboardPage> {
  late Future<List<Workspace>> _future;

  @override
  void initState() {
    super.initState();
    _future = context.read<Session>().api.listWorkspaces();
  }

  Future<void> _reload() async {
    setState(() => _future = context.read<Session>().api.listWorkspaces());
  }

  Future<void> _create() async {
    final l10n = context.read<Appearance>();
    final title = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.t('workspace.createTitle')),
        content: TextField(controller: title, decoration: InputDecoration(labelText: l10n.t('workspace.createNamePlaceholder'))),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('workspace.create'))),
        ],
      ),
    );
    if (ok == true && title.text.trim().isNotEmpty && mounted) {
      await context.read<Session>().api.createWorkspace(title.text.trim(), '');
      await _reload();
    }
    title.dispose();
  }

  Future<void> _join() async {
    final l10n = context.read<Appearance>();
    final code = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.t('workspace.joinTitle')),
        content: TextField(controller: code, decoration: InputDecoration(labelText: l10n.t('workspace.joinCodePlaceholder'))),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('workspace.join'))),
        ],
      ),
    );
    if (ok == true && code.text.trim().isNotEmpty && mounted) {
      await context.read<Session>().api.joinWorkspace(code.text.trim());
      await _reload();
    }
    code.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return AppScaffold(
      title: l10n.t('workspace.title'),
      fab: FloatingActionButton(
        onPressed: _create,
        child: const Icon(Icons.add),
      ),
      body: FutureBuilder(
        future: _future,
        builder: (context, snap) {
          if (snap.hasError) return Center(child: Text('${snap.error}'));
          if (!snap.hasData) return const Center(child: CircularProgressIndicator());
          final list = snap.data!;
          if (list.isEmpty) {
            return Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(l10n.t('workspace.empty')),
                  TextButton(onPressed: _join, child: Text(l10n.t('workspace.join'))),
                ],
              ),
            );
          }
          return ListView(
            children: [
              ListTile(
                title: Text(l10n.t('workspace.joinTitle')),
                leading: const Icon(Icons.group_add_outlined),
                onTap: _join,
              ),
              ...list.map(
                (w) => ListTile(
                  title: Text(w.title),
                  subtitle: Text('${l10n.t('workspace.role.${w.myRole}')} · ${l10n.t('workspace.members', {'count': w.memberCount})}'),
                  onTap: () => context.go('/workspace/${w.id}'),
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}
