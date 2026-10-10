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
import 'roadmap_canvas.dart';
import 'roadmap_header.dart';
import 'roadmap_node_card.dart';
import 'roadmap_progress_view.dart';
import 'roadmap_tree.dart';

class RoadmapPage extends StatefulWidget {
  const RoadmapPage({super.key, required this.projectId});

  final String projectId;

  @override
  State<RoadmapPage> createState() => _RoadmapPageState();
}

enum _RoadmapView { progress, bands, canvas }

class _RoadmapPageState extends State<RoadmapPage> {
  late Future<List<RoadmapNode>> _future;
  late Future<RoadmapProgress> _progressFuture;
  RoadmapProgress? _liveProgress;
  _RoadmapView _view = _RoadmapView.progress;
  final _canvasController = ScrollController();
  List<PresenceEntry> _presence = [];
  final Map<String, CursorDot> _cursors = {};
  DateTime _lastCursorSent = DateTime.fromMillisecondsSinceEpoch(0);
  Timer? _retryTimer;
  late final String _clientId = DateTime.now().microsecondsSinceEpoch
      .toRadixString(36);

  Future<void> _backoff(Duration d) {
    final done = Completer<void>();
    _retryTimer = Timer(d, done.complete);
    return done.future;
  }

  void _sendCursor(double x, double y) {
    final now = DateTime.now();
    if (now.difference(_lastCursorSent).inMilliseconds < 200) return;
    _lastCursorSent = now;
    context
        .read<Session>()
        .api
        .cursor(widget.projectId, x, y)
        .catchError((_) {});
  }

  Future<void> _progressLoop(ApiClient api) async {
    var backoff = const Duration(milliseconds: 500);
    while (mounted) {
      var fatal = false;
      try {
        await api.progressStream(widget.projectId, (progress) {
          if (!mounted) return;
          setState(() => _liveProgress = progress);
        });
      } on ApiException catch (e) {
        fatal = e.status == 401 || e.status == 403 || e.status == 404;
      } catch (_) {}
      if (fatal || !mounted) return;
      await _backoff(backoff);
      backoff = backoff * 2 > const Duration(seconds: 5)
          ? const Duration(seconds: 5)
          : backoff * 2;
    }
  }

  Future<void> _presenceLoop(ApiClient api) async {
    var backoff = const Duration(milliseconds: 500);
    while (mounted) {
      var fatal = false;
      try {
        await api.presenceStream(
          widget.projectId,
          _clientId,
          (entries) {
            if (!mounted) return;
            final live = entries.map((e) => e.sessionId).toSet();
            _cursors.removeWhere((id, _) => !live.contains(id));
            setState(() => _presence = entries);
          },
          onCursor: (sessionId, name, x, y) {
            if (!mounted || sessionId == _clientId) return;
            setState(
              () => _cursors[sessionId] = CursorDot(
                sessionId: sessionId,
                name: name,
                x: x,
                y: y,
              ),
            );
          },
        );
      } on ApiException catch (e) {
        fatal = e.status == 401 || e.status == 403 || e.status == 404;
      } catch (_) {}
      if (fatal || !mounted) return;
      await _backoff(backoff);
      backoff = backoff * 2 > const Duration(seconds: 5)
          ? const Duration(seconds: 5)
          : backoff * 2;
    }
  }

  @override
  void dispose() {
    _retryTimer?.cancel();
    _canvasController.dispose();
    super.dispose();
  }

  @override
  void initState() {
    super.initState();
    context.read<LoopFocus>().rememberProject(widget.projectId);
    _future = context.read<Session>().api.listRoadmap(widget.projectId);
    _progressFuture = context.read<Session>().api.roadmapProgress(
      widget.projectId,
    );
    _progressLoop(context.read<Session>().api);
    _presenceLoop(context.read<Session>().api);
  }

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
    final nodes = context.read<Session>().api.listRoadmap(widget.projectId);
    final progress = context.read<Session>().api.roadmapProgress(
      widget.projectId,
    );
    setState(() {
      _future = nodes;
      _progressFuture = progress;
      _liveProgress = null;
    });
    await Future.wait([nodes, progress]);
  }

  Future<void> _addRoot() async {
    final t = context.read<Appearance>().t;
    final values = await showBlockForm(
      context: context,
      title: t('roadmap.addRoot'),
      confirmLabel: t('roadmap.add'),
      fields: [BlockFormField(label: t('roadmap.rootPlaceholder'))],
    );
    if (values == null || !mounted) return;
    final title = values.first.trim();
    if (title.isEmpty) return;
    final done = await guardWrite(
      context,
      () => context.read<Session>().api.createRoadmapNode(
        widget.projectId,
        title,
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
          RoadmapHeader(compact: narrow, viewChoices: choices, onAdd: _addRoot),
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
                        return snap.hasError
                            ? BlockError(error: snap.error, onRetry: _reload)
                            : const BlockLoading();
                      }
                      return RoadmapProgressView(
                        progress: progress,
                        emptyLabel: l10n.t('roadmap.empty'),
                      );
                    },
                  )
                : FutureBuilder(
                    future: _future,
                    builder: (context, snap) {
                      if (!snap.hasData) {
                        return snap.hasError
                            ? BlockError(error: snap.error, onRetry: _reload)
                            : const BlockLoading();
                      }
                      if (snap.data!.isEmpty) {
                        return BlockEmpty(
                          icon: Icons.account_tree_outlined,
                          body: l10n.t('roadmap.empty'),
                        );
                      }
                      if (_view == _RoadmapView.bands) {
                        return RoadmapBands(
                          projectId: widget.projectId,
                          nodes: snap.data!,
                          onReload: _reload,
                          progress: _liveProgress,
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
