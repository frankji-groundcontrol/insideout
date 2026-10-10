import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/loop_focus.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_form.dart';
import '../../theme/block_state.dart';
import '../../widgets/guarded_action.dart';
import 'home_field.dart';

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
    _future = _fetch();
  }

  Future<List<Workspace>> _fetch() =>
      context.read<Session>().api.listWorkspaces();

  Future<void> _reload() async {
    setState(() {
      _future = _fetch();
    });
  }

  Future<void> _create() async {
    final t = context.read<Appearance>().t;
    final values = await showBlockForm(
      context: context,
      title: t('workspace.createTitle'),
      confirmLabel: t('workspace.create'),
      fields: [BlockFormField(label: t('workspace.createNamePlaceholder'))],
    );
    final title = values?.first.trim() ?? '';
    if (title.isEmpty || !mounted) return;
    final done = await guardWrite(
      context,
      () => context.read<Session>().api.createWorkspace(title, ''),
    );
    if (done && mounted) await _reload();
  }

  Future<void> _join() async {
    final t = context.read<Appearance>().t;
    final values = await showBlockForm(
      context: context,
      title: t('workspace.joinTitle'),
      confirmLabel: t('workspace.join'),
      fields: [BlockFormField(label: t('workspace.joinCodePlaceholder'))],
    );
    final code = values?.first.trim() ?? '';
    if (code.isEmpty || !mounted) return;
    final done = await guardWrite(
      context,
      () => context.read<Session>().api.joinWorkspace(code),
    );
    if (done && mounted) await _reload();
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
      body: FutureBuilder<List<Workspace>>(
        future: _future,
        builder: (context, snap) {
          if (!snap.hasData) return readGate(snap, _reload);
          final list = snap.data!;
          return HomeField(
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
