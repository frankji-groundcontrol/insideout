import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/errors_l10n.dart';
import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/loop_focus.dart';
import '../../chrome/product_dock.dart';
import '../../chrome/time_ago.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/field_card.dart';
import '../../widgets/guarded_action.dart';
import 'ideas_page.dart';

// N1: the square quick-capture FAB floats over the dock's top-right corner.
// The body extends behind the dock (AppScaffold extendBody), so reserve the
// whole FAB zone as scrollable inset — dock + FAB + the scaffold's FAB
// margin + a breathing gap — or the last project card's date row slides
// under the button.
const double _fabSize = 56;
const double _bottomInset =
    ProductDock.height + _fabSize + kFloatingActionButtonMargin + 16;

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

  Future<void> _createProject() async {
    final l10n = context.read<Appearance>();
    final title = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.t('project.newProject')),
        content: TextField(
          controller: title,
          decoration: InputDecoration(
            labelText: l10n.t('project.namePlaceholder'),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: Text(l10n.t('common.cancel')),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(l10n.t('project.create')),
          ),
        ],
      ),
    );
    if (ok == true && title.text.trim().isNotEmpty && mounted) {
      final done = await guardWrite(
        context,
        () => context.read<Session>().api.createProject(
          widget.id,
          title.text.trim(),
          '',
        ),
      );
      if (done && mounted) {
        setState(() {
          _future = _load();
        });
      }
    }
    // Not disposed: the pop animation keeps the dialog's TextField mounted
    // past this await, and the guard's SnackBar rebuilds the overlay — a
    // disposed controller would throw. GC'd once the dialog unmounts.
  }

  Future<void> _refresh() async {
    final future = _load();
    setState(() {
      _future = future;
    });
    await future;
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
          fab: FloatingActionButton(
            onPressed: _createProject,
            child: const Icon(Icons.add),
          ),
          body: !snap.hasData
              ? Center(
                  child: snap.hasError
                      ? Text(localizedError(snap.error!, l10n.t))
                      : const CircularProgressIndicator(),
                )
              : RefreshIndicator(
                  onRefresh: _refresh,
                  child: ListView(
                    padding: const EdgeInsets.fromLTRB(24, 8, 24, _bottomInset),
                    children: [
                      FieldCard(
                        child: Row(
                          children: [
                            Expanded(
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(
                                    l10n.t('workspace.inviteCode', {
                                      'code': snap.data!.ws.code,
                                    }),
                                    style: theme.textTheme.titleMedium,
                                  ),
                                  Text(
                                    l10n.t('workspace.inviteHint'),
                                    style: theme.textTheme.bodySmall,
                                  ),
                                ],
                              ),
                            ),
                            IconButton(
                              icon: const Icon(Icons.copy),
                              onPressed: () => Clipboard.setData(
                                ClipboardData(text: snap.data!.ws.code),
                              ),
                            ),
                          ],
                        ),
                      ),
                      const SizedBox(height: 12),
                      FieldCard(
                        onTap: () =>
                            context.go('/workspace/${widget.id}/ideas'),
                        child: Text(
                          l10n.t('workspace.ideas'),
                          style: theme.textTheme.titleMedium,
                        ),
                      ),
                      const SizedBox(height: 12),
                      FieldCard(
                        onTap: () =>
                            context.go('/workspace/${widget.id}/settings'),
                        child: Text(
                          l10n.t('workspace.settings'),
                          style: theme.textTheme.titleMedium,
                        ),
                      ),
                      const SizedBox(height: 12),
                      FieldCard(
                        onTap: () => _edit(snap.data!.ws),
                        child: Text(
                          l10n.t('common.edit'),
                          style: theme.textTheme.titleMedium,
                        ),
                      ),
                      const SizedBox(height: 12),
                      FieldCard(
                        onTap: _delete,
                        child: Text(
                          l10n.t('workspace.deleteWorkspace'),
                          style: theme.textTheme.titleMedium,
                        ),
                      ),
                      const SizedBox(height: 24),
                      FieldCard(
                        onTap: _createProject,
                        child: Row(
                          children: [
                            const Icon(Icons.add),
                            const SizedBox(width: 12),
                            Text(
                              l10n.t('project.newProject'),
                              style: theme.textTheme.titleMedium,
                            ),
                          ],
                        ),
                      ),
                      const SizedBox(height: 12),
                      if (snap.data!.projects.isEmpty)
                        Text(
                          l10n.t('project.empty'),
                          style: theme.textTheme.bodyLarge,
                        )
                      else
                        for (final p in snap.data!.projects) ...[
                          FieldCard(
                            onTap: () {
                              context.read<LoopFocus>().rememberProject(
                                p.id,
                                workspaceId: widget.id,
                              );
                              context.go('/projects/${p.id}');
                            },
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Row(
                                  children: [
                                    Expanded(
                                      child: Text(
                                        p.title,
                                        style: theme.textTheme.titleLarge,
                                      ),
                                    ),
                                    if (p.flagged)
                                      ShadePill(label: l10n.t('project.quiet')),
                                  ],
                                ),
                                Text(
                                  l10n.t('project.status.${p.status}'),
                                  style: theme.textTheme.bodySmall,
                                ),
                                FutureBuilder<RoadmapProgress>(
                                  future: progressFutureFor(p.id),
                                  builder: (context, psnap) {
                                    if (!psnap.hasData)
                                      return const SizedBox.shrink();
                                    return Padding(
                                      padding: const EdgeInsets.only(top: 4),
                                      child: Text(
                                        l10n.t('workspace.nowDone', {
                                          'now': psnap.data!.now.length,
                                          'done': psnap.data!.doneCount,
                                        }),
                                        style: theme.textTheme.bodySmall,
                                      ),
                                    );
                                  },
                                ),
                                if (p.latestUpdateKind != null)
                                  Padding(
                                    padding: const EdgeInsets.only(top: 8),
                                    child: Row(
                                      children: [
                                        Text(
                                          l10n.t(
                                            'project.updateKind.${p.latestUpdateKind}',
                                          ),
                                          style: theme.textTheme.labelSmall
                                              ?.copyWith(
                                                fontWeight: FontWeight.w700,
                                              ),
                                        ),
                                        const SizedBox(width: 8),
                                        Expanded(
                                          child: Text(
                                            p.latestUpdateContent ?? '',
                                            maxLines: 1,
                                            overflow: TextOverflow.ellipsis,
                                            style: theme.textTheme.bodySmall,
                                          ),
                                        ),
                                        if (p.latestUpdateAt != null)
                                          Padding(
                                            padding: const EdgeInsets.only(
                                              left: 8,
                                            ),
                                            child: Text(
                                              timeAgo(
                                                p.latestUpdateAt!,
                                                l10n.t,
                                              ),
                                              style: theme.textTheme.bodySmall,
                                            ),
                                          ),
                                      ],
                                    ),
                                  ),
                              ],
                            ),
                          ),
                          const SizedBox(height: 12),
                        ],
                    ],
                  ),
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
            TextField(
              controller: title,
              decoration: InputDecoration(
                labelText: l10n.t('workspace.createNamePlaceholder'),
              ),
            ),
            TextField(
              controller: desc,
              decoration: InputDecoration(
                labelText: l10n.t('workspace.createDescPlaceholder'),
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
            child: Text(l10n.t('common.save')),
          ),
        ],
      ),
    );
    if (ok == true && mounted) {
      final done = await guardWrite(
        context,
        () => context.read<Session>().api.updateWorkspace(
          widget.id,
          title: title.text.trim(),
          description: desc.text,
        ),
      );
      if (done && mounted) {
        setState(() {
          _future = _load();
        });
      }
    }
    // Not disposed: the pop animation keeps the dialog's TextFields mounted
    // past this await, and the guard's SnackBar rebuilds the overlay — a
    // disposed controller would throw. GC'd once the dialog unmounts.
  }

  Future<void> _delete() async {
    final l10n = context.read<Appearance>();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.t('workspace.deleteWorkspaceConfirm')),
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
      final done = await guardWrite(
        context,
        () => context.read<Session>().api.deleteWorkspace(widget.id),
      );
      if (done && mounted) context.go('/dashboard');
    }
  }
}
