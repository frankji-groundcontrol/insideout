import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/loop_focus.dart';
import '../../chrome/product_dock.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_action_card.dart';
import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/block_form.dart';
import '../../theme/block_modal.dart';
import '../../theme/block_state.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_theme.dart';
import '../../theme/field_card.dart';
import '../../widgets/guarded_action.dart';
import 'block_refresh.dart';
import 'board_project_card.dart';
import 'quick_capture_fab.dart';

// N1: the square quick-capture FAB floats over the dock's top-right corner.
// The body extends behind the dock (AppScaffold extendBody), so reserve the
// whole FAB zone as scrollable inset — dock + FAB + the scaffold's FAB
// margin + a breathing gap — or the last project card's date row slides
// under the button.
const double _bottomInset =
    ProductDock.height +
    quickCaptureSize +
    kFloatingActionButtonMargin +
    Cobalt.space4;

class WorkspacePage extends StatefulWidget {
  const WorkspacePage({super.key, required this.id});

  final String id;

  @override
  State<WorkspacePage> createState() => _WorkspacePageState();
}

class _WorkspacePageState extends State<WorkspacePage> {
  final Map<String, Future<RoadmapProgress>> _progressFutures = {};

  /// One progress fetch per project, memoized so list rebuilds do not
  /// refetch.
  Future<RoadmapProgress> progressFutureFor(String projectId) {
    return _progressFutures.putIfAbsent(
      projectId,
      () => context.read<Session>().api.roadmapProgress(projectId),
    );
  }

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
    if (!mounted) return (ws: ws, projects: projects);
    final focus = context.read<LoopFocus>();
    if (focus.projectFor(widget.id) == null && projects.isNotEmpty) {
      focus.rememberProject(projects.first.id, workspaceId: widget.id);
    }
    return (ws: ws, projects: projects);
  }

  Future<void> _reload() async {
    setState(() {
      _future = _load();
    });
    await _future;
  }

  Future<void> _createProject() async {
    final t = context.read<Appearance>().t;
    final values = await showBlockForm(
      context: context,
      title: t('project.newProject'),
      confirmLabel: t('project.create'),
      fields: [
        BlockFormField(label: t('project.namePlaceholder'), required: true),
      ],
    );
    final title = values?.first.trim() ?? '';
    if (title.isEmpty || !mounted) return;
    final done = await guardWrite(
      context,
      () => context.read<Session>().api.createProject(widget.id, title, ''),
    );
    if (done && mounted) await _reload();
  }

  Future<void> _edit(Workspace ws) async {
    final t = context.read<Appearance>().t;
    final values = await showBlockForm(
      context: context,
      title: t('common.edit'),
      confirmLabel: t('common.save'),
      fields: [
        BlockFormField(
          label: t('workspace.createNamePlaceholder'),
          initial: ws.title,
          required: true,
        ),
        BlockFormField(
          label: t('workspace.createDescPlaceholder'),
          initial: ws.description,
          multiline: true,
        ),
      ],
    );
    if (values == null || !mounted) return;
    final done = await guardWrite(
      context,
      () => context.read<Session>().api.updateWorkspace(
        widget.id,
        title: values[0].trim(),
        description: values[1],
      ),
    );
    if (done && mounted) await _reload();
  }

  Future<void> _delete() async {
    final t = context.read<Appearance>().t;
    final ok = await showBlockConfirm(
      context: context,
      title: t('workspace.deleteWorkspace'),
      message: t('workspace.deleteWorkspaceConfirm'),
      confirmText: t('common.delete'),
      danger: true,
    );
    if (!ok || !mounted) return;
    final done = await guardWrite(
      context,
      () => context.read<Session>().api.deleteWorkspace(widget.id),
    );
    if (done && mounted) context.go('/dashboard');
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<({Workspace ws, List<Project> projects})>(
      future: _future,
      builder: (context, snap) {
        final l10n = context.watch<Appearance>();
        final title = snap.data?.ws.title ?? l10n.t('workspace.title');
        return AppScaffold(
          title: title,
          paintTitle: true,
          fab: QuickCaptureFab(
            label: l10n.t('project.newProject'),
            onPressed: _createProject,
          ),
          body: !snap.hasData
              ? readGate(snap, _reload)
              : BlockRefresh(
                  label: l10n.t('common.refresh'),
                  onRefresh: _reload,
                  child: _board(context, snap.data!),
                ),
        );
      },
    );
  }

  Widget _board(
    BuildContext context,
    ({Workspace ws, List<Project> projects}) data,
  ) {
    final l10n = context.watch<Appearance>();
    final width = MediaQuery.sizeOf(context).width;
    final hPad = width >= Cobalt.bpDesk
        ? Cobalt.space8
        : width >= Cobalt.bpNarrow
        ? Cobalt.space6
        : Cobalt.space4;
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: Cobalt.contentMaxWidth),
        child: ListView(
          padding: EdgeInsets.fromLTRB(hPad, Cobalt.space2, hPad, _bottomInset),
          children: [
            _inviteCard(context, data.ws),
            const SizedBox(height: Cobalt.space3),
            BlockActionCard(
              icon: Icons.lightbulb_outline,
              label: l10n.t('workspace.ideas'),
              onTap: () => context.go('/workspace/${widget.id}/ideas'),
            ),
            const SizedBox(height: Cobalt.space3),
            BlockActionCard(
              icon: Icons.settings_outlined,
              label: l10n.t('workspace.settings'),
              onTap: () => context.go('/workspace/${widget.id}/settings'),
            ),
            const SizedBox(height: Cobalt.space4),
            Wrap(
              spacing: Cobalt.space3,
              runSpacing: Cobalt.space3,
              children: [
                BlockButton(
                  label: l10n.t('common.edit'),
                  icon: Icons.edit_outlined,
                  variant: BlockButtonVariant.secondary,
                  onPressed: () => _edit(data.ws),
                ),
                BlockButton(
                  label: l10n.t('workspace.deleteWorkspace'),
                  icon: Icons.delete_outline,
                  variant: BlockButtonVariant.danger,
                  onPressed: _delete,
                ),
              ],
            ),
            const SizedBox(height: Cobalt.space8),
            if (data.projects.isEmpty)
              BlockEmpty(
                icon: Icons.account_tree_outlined,
                title: l10n.t('project.title'),
                body: l10n.t('project.empty'),
                action: BlockButton(
                  label: l10n.t('project.newProject'),
                  icon: Icons.add,
                  onPressed: _createProject,
                ),
              )
            else ...[
              Row(
                children: [
                  Text(
                    l10n.t('project.title').toUpperCase(),
                    style: eyebrow(context),
                  ),
                  const Spacer(),
                  BlockButton(
                    label: l10n.t('project.newProject'),
                    icon: Icons.add,
                    size: BlockButtonSize.sm,
                    onPressed: _createProject,
                  ),
                ],
              ),
              const SizedBox(height: Cobalt.space3),
              for (final p in data.projects) ...[
                BoardProjectCard(
                  project: p,
                  progress: progressFutureFor(p.id),
                  onOpen: () {
                    context.read<LoopFocus>().rememberProject(
                      p.id,
                      workspaceId: widget.id,
                    );
                    context.go('/projects/${p.id}');
                  },
                ),
                const SizedBox(height: Cobalt.space3),
              ],
            ],
          ],
        ),
      ),
    );
  }

  Widget _inviteCard(BuildContext context, Workspace ws) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    return FieldCard(
      padding: const EdgeInsets.all(Cobalt.space4),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  l10n.t('workspace.inviteHint').toUpperCase(),
                  style: eyebrow(context),
                ),
                const SizedBox(height: Cobalt.space1),
                Text(
                  l10n.t('workspace.inviteCode', {'code': ws.code}),
                  style: theme.textTheme.titleMedium,
                ),
                const SizedBox(height: Cobalt.space2),
                Text(
                  l10n.t('workspace.inviteAccountHint'),
                  style: theme.textTheme.bodySmall,
                ),
              ],
            ),
          ),
          BlockIconButton(
            icon: Icons.copy,
            tooltip: l10n.t('common.copy'),
            onPressed: () => Clipboard.setData(ClipboardData(text: ws.code)),
          ),
        ],
      ),
    );
  }
}
