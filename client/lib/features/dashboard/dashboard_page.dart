import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/loop_focus.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/cobalt.dart';
import '../../theme/field_card.dart';

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
    setState(() {
      _future = context.read<Session>().api.listWorkspaces();
    });
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

  void _openWorkspace(Workspace w) {
    context.read<LoopFocus>().rememberWorkspace(w.id);
    context.go('/workspace/${w.id}');
  }

  void _capture(List<Workspace> list) {
    if (list.isEmpty) {
      _create();
      return;
    }
    context.read<LoopFocus>().rememberWorkspace(list.first.id);
    context.go('/workspace/${list.first.id}/ideas');
  }

  @override
  Widget build(BuildContext context) {
    context.watch<Appearance>();
    return AppScaffold(
      body: FutureBuilder(
        future: _future,
        builder: (context, snap) {
          if (snap.hasError) return Center(child: Text('${snap.error}'));
          if (!snap.hasData) return const Center(child: CircularProgressIndicator());
          final list = snap.data!;
          return _HomeField(
            workspaces: list,
            onCreate: _create,
            onJoin: _join,
            onOpen: _openWorkspace,
            onCapture: () => _capture(list),
          );
        },
      ),
    );
  }
}

class _HomeField extends StatelessWidget {
  const _HomeField({
    required this.workspaces,
    required this.onCreate,
    required this.onJoin,
    required this.onOpen,
    required this.onCapture,
  });

  final List<Workspace> workspaces;
  final VoidCallback onCreate;
  final VoidCallback onJoin;
  final void Function(Workspace w) onOpen;
  final VoidCallback onCapture;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    final wide = MediaQuery.sizeOf(context).width >= 900;
    final numeral = workspaces.isEmpty ? '00' : workspaces.length.toString().padLeft(2, '0');
    return ListView(
      padding: const EdgeInsets.fromLTRB(28, 72, 28, 140),
      children: [
        Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(l10n.t('home.loopTitle').toUpperCase(), style: theme.textTheme.displaySmall),
                  const SizedBox(height: 8),
                  Text(l10n.t('home.loopLead'), style: theme.textTheme.bodyLarge),
                ],
              ),
            ),
            FilledButton.icon(
              onPressed: onCapture,
              icon: const Icon(Icons.arrow_forward, size: 18),
              label: Text(l10n.t('home.captureIdea')),
            ),
          ],
        ),
        const SizedBox(height: 28),
        if (wide)
          IntrinsicHeight(
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Expanded(child: _IndexCard(numeral: numeral)),
                const SizedBox(width: 16),
                Expanded(child: _WorkspaceColumn(workspaces: workspaces, onOpen: onOpen, onJoin: onJoin, onCreate: onCreate)),
              ],
            ),
          )
        else ...[
          _IndexCard(numeral: numeral),
          const SizedBox(height: 16),
          _WorkspaceColumn(workspaces: workspaces, onOpen: onOpen, onJoin: onJoin, onCreate: onCreate),
        ],
        const SizedBox(height: 24),
        Text(l10n.t('home.loopTrail'), style: theme.textTheme.labelSmall?.copyWith(color: palette.fgBrand)),
      ],
    );
  }
}

class _IndexCard extends StatelessWidget {
  const _IndexCard({required this.numeral});

  final String numeral;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    return FieldCard(
      color: palette.surfaceSunken,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(l10n.t('home.chapterEyebrow', {'n': numeral}), style: theme.textTheme.labelSmall?.copyWith(color: palette.fgBrand)),
          const SizedBox(height: 12),
          Text(l10n.t('home.chapterTitle'), style: theme.textTheme.headlineLarge),
          const SizedBox(height: 8),
          Text(
            numeral,
            style: theme.textTheme.displayLarge?.copyWith(color: palette.brand, fontSize: 120, height: 0.9),
          ),
          const SizedBox(height: 12),
          Text(l10n.t('home.chapterBody'), style: theme.textTheme.bodyLarge),
        ],
      ),
    );
  }
}

class _WorkspaceColumn extends StatelessWidget {
  const _WorkspaceColumn({
    required this.workspaces,
    required this.onOpen,
    required this.onJoin,
    required this.onCreate,
  });

  final List<Workspace> workspaces;
  final void Function(Workspace w) onOpen;
  final VoidCallback onJoin;
  final VoidCallback onCreate;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    if (workspaces.isEmpty) {
      return FieldCard(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(l10n.t('workspace.empty'), style: theme.textTheme.bodyLarge),
            const SizedBox(height: 16),
            Wrap(
              spacing: 8,
              children: [
                FilledButton(onPressed: onCreate, child: Text(l10n.t('workspace.create'))),
                OutlinedButton(onPressed: onJoin, child: Text(l10n.t('workspace.join'))),
              ],
            ),
          ],
        ),
      );
    }
    return Column(
      children: [
        FieldCard(
          onTap: onJoin,
          child: Row(
            children: [
              const Icon(Icons.group_add_outlined),
              const SizedBox(width: 12),
              Expanded(child: Text(l10n.t('workspace.joinTitle'), style: theme.textTheme.titleMedium)),
              const Icon(Icons.chevron_right),
            ],
          ),
        ),
        for (var i = 0; i < workspaces.length; i++) ...[
          const SizedBox(height: 12),
          FieldCard(
            onTap: () => onOpen(workspaces[i]),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Text(
                      (i + 1).toString().padLeft(2, '0'),
                      style: theme.textTheme.displaySmall?.copyWith(fontSize: 48, height: 1),
                    ),
                    const Spacer(),
                    const Icon(Icons.chevron_right),
                  ],
                ),
                const SizedBox(height: 12),
                Text(workspaces[i].title, style: theme.textTheme.titleLarge),
                const SizedBox(height: 6),
                Text(
                  '${l10n.t('workspace.role.${workspaces[i].myRole}')} · ${l10n.t('workspace.members', {'count': workspaces[i].memberCount})}',
                  style: theme.textTheme.bodyLarge,
                ),
              ],
            ),
          ),
        ],
      ],
    );
  }
}
