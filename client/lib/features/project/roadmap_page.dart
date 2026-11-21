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

class RoadmapPage extends StatefulWidget {
  const RoadmapPage({super.key, required this.projectId, this.rootId});

  final String projectId;

  /// The node opened as its own tree (/projects/:id/roadmap/:nodeId): the
  /// page shows it and the nodes below it. Null for the whole project.
  final String? rootId;

  @override
  State<RoadmapPage> createState() => _RoadmapPageState();
}

enum _RoadmapView { progress, bands, canvas }

class _RoadmapPageState extends State<RoadmapPage> with _RoadmapLive {
  late Future<List<RoadmapNode>> _future;
  late Future<RoadmapProgress> _progressFuture;
  Future<RoadmapNodeRead>? _read;
  _RoadmapView _view = _RoadmapView.progress;
  final _canvasController = ScrollController();

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
    _fetch(api);
    api
        .getProject(widget.projectId)
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
          if (mounted) setState(() => _rootWritable = r.node.canWrite);
        })
        .catchError((_) {});
  }

  /// The opened node's canWrite: "New item" adds under it.
  bool _rootWritable = false;

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
          }),
          selected: _view == view,
          onTap: () => setState(() => _view = view),
        ),
    ];
    // Three toggles plus four utility icons run off a phone's left edge, so
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
                ? FutureBuilder(
                    future: _progressFuture,
                    builder: (context, snap) {
                      final progress = _liveProgress ?? snap.data;
                      if (progress == null) {
                        return readGate(snap, _reload);
                      }
                      // Progress lists only dated work, so an empty one does
                      // not mean there are no goals.
                      return FutureBuilder(
                        future: _future,
                        builder: (context, nodes) => RoadmapProgressView(
                          progress: progress,
                          // Blank until the goals are known, so the wrong
                          // sentence never flashes.
                          emptyLabel: nodes.hasData
                              ? l10n.t(
                                  nodes.data!.isEmpty
                                      ? 'roadmap.empty'
                                      : 'roadmap.noneScheduled',
                                )
                              : '',
                        ),
                      );
                    },
                  )
                : FutureBuilder(
                    future: _future,
                    builder: (context, snap) {
                      if (!snap.hasData) {
                        return readGate(snap, _reload);
                      }
                      if (snap.data!.isEmpty) {
                        return BlockEmpty(
                          icon: Icons.account_tree_outlined,
                          body: l10n.t('roadmap.empty'),
                        );
                      }
                      if (_view == _RoadmapView.bands) {
                        // The reading the Progress tab shows, so a reload or
                        // a quiet stream cannot make the two disagree.
                        return FutureBuilder(
                          future: _progressFuture,
                          builder: (context, ps) {
                            final progress = _liveProgress ?? ps.data;
                            if (progress == null && !ps.hasError) {
                              return const BlockLoading();
                            }
                            return RoadmapBands(
                              projectId: widget.projectId,
                              nodes: snap.data!,
                              onReload: _reload,
                              progress: progress,
                              planWriter: _planWriter,
                            );
                          },
                        );
                      }
                      return Column(
                        children: [
                          RoadmapMinimap(
                            nodes: snap.data!,
                            onTapBand: _scrollToBand,
                          ),
                          const Divider(height: 1),
                          Expanded(
                            child: RoadmapCanvas(
                              nodes: snap.data!,
                              tileBuilder: (node, _) => RoadmapNodeCard(
                                projectId: widget.projectId,
                                node: node,
                                onReload: _reload,
                                childProgress: RoadmapNodeCard.childDoneShare(
                                  snap.data!,
                                  node,
                                ),
                                descendants: roadmapDescendantCount(
                                  snap.data!,
                                  node,
                                ),
                                planWriter: _planWriter,
                                parentWritable: roadmapParentWritable(
                                  snap.data!,
                                  node,
                                ),
                              ),
                              controller: _canvasController,
                              onCursor: _sendCursor,
                              cursors: _cursors.values.toList(),
                            ),
                          ),
                        ],
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }
}
