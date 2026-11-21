import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/client.dart';
import '../../api/errors.dart';
import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/loop_focus.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_form.dart';
import '../../theme/block_state.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';
import '../../theme/ink_motion.dart';
import '../../widgets/guarded_action.dart';
import 'roadmap_bands.dart';
import 'roadmap_breadcrumb.dart';
import 'roadmap_canvas.dart';
import 'roadmap_header.dart';
import 'roadmap_node_card.dart';
import 'roadmap_progress_view.dart';
import 'roadmap_tree.dart';

part 'roadmap_live.dart';
part 'roadmap_page_views.dart';

class RoadmapPage extends StatefulWidget {
  const RoadmapPage({super.key, required this.projectId, this.rootId});

  final String projectId;

  /// The node opened as its own tree (/projects/:id/roadmap/:nodeId): the
  /// page shows it and the nodes below it. Null for the whole project.
  final String? rootId;

  @override
  State<RoadmapPage> createState() => _RoadmapPageState();
}

/// Mine is the canvas over the caller's own branches (owner decision 2: a
/// default view, not a wall).
enum _RoadmapView { progress, bands, canvas, mine }

class _RoadmapPageState extends State<RoadmapPage> with _RoadmapLive {
  late Future<List<RoadmapNode>> _future;
  late Future<RoadmapProgress> _progressFuture;
  Future<RoadmapNodeRead>? _read;
  late final Future<Map<String, dynamic>> _project;
  _RoadmapView _view = _RoadmapView.progress;
  final _canvasController = ScrollController();

  /// Branches folded on the canvas, kept for this page session.
  final _collapsed = <String>{};

  void _toggleFold(String nodeId) =>
      setState(() => _collapsed.remove(nodeId) || _collapsed.add(nodeId));

  /// The caller writes the whole plan (GET /projects/{pid} canWritePlan).
  bool _planWriter = false;

  @override
  void dispose() {
    _canvasController.dispose();
    super.dispose();
  }

  @override
  void initState() {
    super.initState();
    final api = context.read<Session>().api;
    context.read<LoopFocus>().rememberProject(widget.projectId);
    _project = api.getProject(widget.projectId);
    _fetch(api);
    _project
        .then((p) {
          if (mounted) setState(() => _planWriter = p['canWritePlan'] == true);
        })
        // A failed read leaves the plan-writer items hidden; the server
        // refuses them anyway.
        .catchError((_) {});
    // The live progress reading is project-wide: a subtree keeps its own.
    if (widget.rootId == null) _progressLoop(api);
    _presenceLoop(api);
  }

  void _fetch(ApiClient api) {
    final root = widget.rootId;
    _future = root == null
        ? api.listRoadmap(widget.projectId)
        : api.listRoadmapSubtree(widget.projectId, root);
    _progressFuture = root == null
        ? api.roadmapProgress(widget.projectId)
        : api.roadmapSubtreeProgress(widget.projectId, root);
    _read = root == null ? null : api.readRoadmapNode(root);
    _read
        ?.then((r) {
          if (!mounted) return;
          setState(() {
            _rootWritable = r.node.canWrite;
            _opened = r;
          });
          if (_names.isEmpty && r.ancestors.any((a) => a.ownerId != null)) {
            _loadNames(api);
          }
        })
        .catchError((_) {});
  }

  /// The opened node's canWrite: "New item" adds under it.
  bool _rootWritable = false;

  /// The opened node's read: its ancestors carry the owners its tree
  /// inherits, for Mine and the inherited owner label.
  RoadmapNodeRead? _opened;

  /// Member names by user id, read only when an ancestor above the opened
  /// tree has an owner (the ancestor refs carry the id alone).
  Map<String, String> _names = const {};

  void _loadNames(ApiClient api) => _project
      .then((p) => api.listMembers(p['workspaceId'] as String))
      .then((members) {
        if (!mounted) return;
        setState(
          () => _names = {for (final m in members) m.userId: m.username},
        );
      })
      // Unnamed, the label reads "someone".
      .catchError((_) {});

  void _scrollToBand(int i) {
    if (!_canvasController.hasClients) return;
    final target =
        (i * (RoadmapCanvas.bandWidth + RoadmapCanvas.bandSpacing) - 24)
            .clamp(0.0, _canvasController.position.maxScrollExtent)
            .toDouble();
    if (inkReduceMotion(context)) {
      _canvasController.jumpTo(target);
      return;
    }
    _canvasController.animateTo(
      target,
      duration: const Duration(milliseconds: 300),
      curve: Curves.easeOutCubic,
    );
  }

  Future<void> _reload() async {
    setState(() {
      _fetch(context.read<Session>().api);
      _liveProgress = null;
    });
    await Future.wait([_future, _progressFuture, ?_read]);
  }

  /// "New item": a top-level goal for a plan writer, or in an opened node a
  /// child of that node for whoever may write it.
  Future<void> _addRoot() async {
    final t = context.read<Appearance>().t;
    final values = await showBlockForm(
      context: context,
      title: t(widget.rootId == null ? 'roadmap.addRoot' : 'roadmap.addChild'),
      confirmLabel: t('roadmap.add'),
      fields: [
        BlockFormField(label: t('roadmap.rootPlaceholder'), required: true),
      ],
    );
    if (values == null || !mounted) return;
    final title = values.first.trim();
    if (title.isEmpty) return;
    final done = await guardWrite(
      context,
      () => context.read<Session>().api.createRoadmapNode(
        widget.projectId,
        title,
        parentId: widget.rootId,
      ),
    );
    if (done && mounted) await _reload();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final choices = [
      for (final view in _RoadmapView.values)
        BlockChoice(
          label: l10n.t(switch (view) {
            _RoadmapView.progress => 'roadmap.viewProgress',
            _RoadmapView.bands => 'roadmap.viewBands',
            _RoadmapView.canvas => 'roadmap.viewCanvas',
            _RoadmapView.mine => 'roadmap.viewMine',
          }),
          selected: _view == view,
          onTap: () => setState(() => _view = view),
        ),
    ];
    // Four toggles plus four utility icons run off a phone's left edge, so
    // there the toggles move into the header.
    final narrow = MediaQuery.sizeOf(context).width < Cobalt.bpNarrow;
    return AppScaffold(
      actions: narrow ? null : choices,
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          RoadmapHeader(
            compact: narrow,
            viewChoices: choices,
            onAdd: (widget.rootId == null ? _planWriter : _rootWritable)
                ? _addRoot
                : null,
            breadcrumb: _read == null
                ? null
                : RoadmapBreadcrumb(projectId: widget.projectId, read: _read!),
          ),
          if (_presence.isNotEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 6),
              child: Wrap(
                spacing: Cobalt.space2,
                children: [
                  Icon(
                    Icons.groups_2_outlined,
                    size: 16,
                    color: Cobalt.of(context).brand,
                  ),
                  for (final e in _presence)
                    ShadeChip(label: e.name, tone: ShadeTone.neutral),
                ],
              ),
            ),
          Expanded(
            child: _view == _RoadmapView.progress
                ? _progressBody(l10n)
                : _treeBody(l10n),
          ),
        ],
      ),
    );
  }
}
