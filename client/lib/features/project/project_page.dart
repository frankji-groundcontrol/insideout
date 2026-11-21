import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/loop_focus.dart';
import '../../chrome/product_dock.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_state.dart';
import '../../theme/cobalt.dart';
import '../../widgets/guarded_action.dart';
import 'project_dialogs.dart';
import 'project_github_card.dart';
import 'project_header.dart';
import 'project_leaf_evidence.dart';
import 'project_updates_feed.dart';

/// Clears the top-right utility block on the field; the body extends behind
/// the dock (AppScaffold extendBody).
const double _topInset = 64;

class ProjectPage extends StatefulWidget {
  const ProjectPage({super.key, required this.id});

  final String id;

  @override
  State<ProjectPage> createState() => _ProjectPageState();
}

class _ProjectPageState extends State<ProjectPage> {
  late Future<ProjectDetail> _future;
  late Future<LeafEvidence> _leafEvidence;
  final repo = TextEditingController();
  final _scroll = ScrollController();
  Timer? _flashTimer;
  var _flash = false;
  var _lastPulse = 0;
  late final LoopFocus _focus;

  /// The workspace members: the owner's name and the edit dialog's picker.
  List<WorkspaceMember> _members = const [];

  /// Only an admin hands a project over (the server refuses anyone else).
  bool get _isAdmin {
    final me = context.read<Session>().user?.id;
    return _members.any((m) => m.userId == me && m.role == 'admin');
  }

  @override
  void initState() {
    super.initState();
    _focus = context.read<LoopFocus>();
    _focus.rememberProject(widget.id);
    // The dock bumps evidencePulse when its Evidence step is tapped while
    // this page is already the destination; one-shot per bump (the counter
    // also filters the focus's other notifications).
    _lastPulse = _focus.evidencePulse;
    _focus.addListener(_onEvidencePulse);
    _future = _load();
    _leafEvidence = _loadLeafEvidence()..ignore();
  }

  @override
  void dispose() {
    // Already deactivated: context.read would throw and skip the timer cancel.
    _flashTimer?.cancel();
    _focus.removeListener(_onEvidencePulse);
    _scroll.dispose();
    repo.dispose();
    super.dispose();
  }

  /// Pulse from the dock's Evidence item: scroll the evidence header back
  /// into view and flash it, so the tap reads as "you are here".
  void _onEvidencePulse() {
    if (_focus.evidencePulse == _lastPulse) return;
    _lastPulse = _focus.evidencePulse;
    if (!mounted) return;
    setState(() => _flash = true);
    _flashTimer?.cancel();
    _flashTimer = Timer(const Duration(seconds: 1), () {
      if (mounted) setState(() => _flash = false);
    });
    if (_scroll.hasClients) {
      // A ticker animation does not move on the pump that shows the flash.
      _scroll.jumpTo(0);
    }
  }

  Future<ProjectDetail> _load() async {
    final api = context.read<Session>().api;
    final data = await api.getProject(widget.id);
    if (!mounted) return data;
    final ws = data.project.workspaceId;
    _focus.rememberProject(widget.id, workspaceId: ws);
    repo.text = data.project.repoUrl;
    // The page still works without names; the owner line just stays empty.
    final members = await api.listMembers(ws).catchError((_) => _members);
    if (mounted) _members = members;
    return data;
  }

  /// The only per-node evidence route is one request per leaf, so the section
  /// lists this many finished leaves and says how many it left out.
  static const _leafEvidenceCap = 5;

  /// Evidence rows for the finished leaves, so the Evidence screen shows what
  /// finished work actually produced. A failure is left to reach the
  /// section's FutureBuilder, which says so instead of showing nothing. Callers
  /// `ignore()` the future: that builder only mounts once the project itself
  /// has loaded, and an error landing earlier would escape as unhandled.
  Future<LeafEvidence> _loadLeafEvidence() async {
    final api = context.read<Session>().api;
    final nodes = await api.listRoadmap(widget.id);
    // A validated leaf is finished work plus its confirm — its evidence
    // (the observation that justified validation) stays listed.
    final done = nodes
        .where(
          (n) =>
              n.parentId != null &&
              (n.status == 'done' || n.status == 'validated'),
        )
        .toList();
    final leaves = await Future.wait(
      done.take(_leafEvidenceCap).map((n) async {
        final rows = await api.listNodeEvidence(n.id);
        return (title: n.title, rows: rows);
      }),
    );
    return (leaves: leaves, more: done.length - leaves.length);
  }

  Future<void> _reload() async {
    setState(() {
      _future = _load();
      _leafEvidence = _loadLeafEvidence()..ignore();
    });
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder(
      future: _future,
      builder: (context, snap) {
        final l10n = context.watch<Appearance>();
        final project = snap.data?.project;
        final title = project?.title ?? l10n.t('evidence.title');
        return AppScaffold(
          body: !snap.hasData
              ? readGate(snap, _reload)
              : Center(
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(
                      maxWidth: Cobalt.contentMaxWidth,
                    ),
                    child: ListView(
                      controller: _scroll,
                      padding: const EdgeInsets.fromLTRB(
                        28,
                        _topInset,
                        28,
                        ProductDock.clearance,
                      ),
                      children: [
                        ProjectHeader(
                          title: title,
                          ownerName: projectOwnerName(
                            project!.ownerId,
                            _members,
                          ),
                          flash: _flash,
                        ),
                        const SizedBox(height: 20),
                        Wrap(
                          spacing: Cobalt.space2,
                          runSpacing: Cobalt.space2,
                          children: [
                            BlockButton(
                              label: l10n.t('roadmap.title'),
                              icon: Icons.account_tree_outlined,
                              onPressed: () =>
                                  context.go('/projects/${widget.id}/roadmap'),
                            ),
                            if (project.prdId case final String prdId)
                              BlockButton(
                                label: l10n.t('project.openPrd'),
                                icon: Icons.description_outlined,
                                variant: BlockButtonVariant.secondary,
                                onPressed: () {
                                  _focus.rememberPrd(prdId);
                                  context.go('/prd/$prdId');
                                },
                              ),
                            if (project.canWritePlan)
                              BlockButton(
                                label: l10n.t('project.edit'),
                                icon: Icons.edit_outlined,
                                variant: BlockButtonVariant.secondary,
                                onPressed: () => _edit(project),
                              ),
                            // Deleting a project is an admin's alone.
                            if (_isAdmin)
                              BlockButton(
                                label: l10n.t('project.delete'),
                                icon: Icons.delete_outline,
                                variant: BlockButtonVariant.danger,
                                onPressed: () => _delete(project),
                              ),
                          ],
                        ),
                        const SizedBox(height: 20),
                        ProjectGithubCard(
                          projectId: widget.id,
                          repo: repo,
                          onChanged: _reload,
                        ),
                        ProjectLeafEvidence(future: _leafEvidence),
                        const SizedBox(height: 20),
                        ProjectUpdatesFeed(
                          projectId: widget.id,
                          updates: snap.data!.updates,
                          nextCursor: snap.data!.nextCursor,
                          canDecide: project.canWritePlan,
                          onChanged: _reload,
                          onEdit: (u, content) => context
                              .read<Session>()
                              .api
                              .editUpdate(u.id, content),
                          onRemove: (u) =>
                              context.read<Session>().api.removeUpdate(u.id),
                        ),
                        BlockButton(
                          label: l10n.t('project.addUpdate'),
                          icon: Icons.add,
                          expand: true,
                          onPressed: _addUpdate,
                        ),
                      ],
                    ),
                  ),
                ),
        );
      },
    );
  }

  Future<void> _edit(Project data) async {
    final saved = await showProjectEdit(
      context,
      data,
      members: _isAdmin ? _members : null,
    );
    if (saved == null || !mounted) return;
    final done = await guardWrite(
      context,
      () => context.read<Session>().api.updateProject(
        widget.id,
        title: saved.title,
        description: saved.description,
        status: saved.status,
        // Sent only when changed (PARB-08); '' hands the project to no one.
        ownerId: saved.ownerId == (data.ownerId ?? '') ? null : saved.ownerId,
      ),
    );
    if (done && mounted) await _reload();
  }

  Future<void> _delete(Project data) async {
    if (!await showProjectDelete(context) || !mounted) return;
    final wsId = data.workspaceId;
    final done = await guardWrite(
      context,
      () => context.read<Session>().api.deleteProject(widget.id),
    );
    if (done && mounted) context.go('/workspace/$wsId');
  }

  Future<void> _addUpdate() async {
    final posted = await showProjectAddUpdate(context);
    if (posted == null || !mounted) return;
    final done = await guardWrite(
      context,
      () => context.read<Session>().api.addUpdate(
        widget.id,
        posted.kind,
        posted.content,
      ),
    );
    if (done && mounted) await _reload();
  }
}
