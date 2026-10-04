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
import '../../theme/cobalt.dart';
import 'roadmap_bands.dart';
import 'roadmap_canvas.dart';
import 'roadmap_node_card.dart';
import 'roadmap_progress_view.dart';

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
  _RoadmapView _view = _RoadmapView.progress;
  final _canvasController = ScrollController();
  List<PresenceEntry> _presence = [];
  final Map<String, CursorDot> _cursors = {};
  DateTime _lastCursorSent = DateTime.fromMillisecondsSinceEpoch(0);
  Timer? _retryTimer;
  late final String _clientId = DateTime.now().microsecondsSinceEpoch.toRadixString(36);

  Future<void> _backoff(Duration d) {
    final done = Completer<void>();
    _retryTimer = Timer(d, done.complete);
    return done.future;
  }

  void _sendCursor(double x, double y) {
    final now = DateTime.now();
    if (now.difference(_lastCursorSent).inMilliseconds < 200) return;
    _lastCursorSent = now;
    context.read<Session>().api.cursor(widget.projectId, x, y).catchError((_) {});
  }

  Future<void> _presenceLoop(ApiClient api) async {
    var backoff = const Duration(milliseconds: 500);
    while (mounted) {
      var fatal = false;
      try {
        await api.presenceStream(widget.projectId, _clientId, (entries) {
          if (!mounted) return;
          final live = entries.map((e) => e.sessionId).toSet();
          _cursors.removeWhere((id, _) => !live.contains(id));
          setState(() => _presence = entries);
        }, onCursor: (sessionId, name, x, y) {
          if (!mounted || sessionId == _clientId) return;
          setState(() => _cursors[sessionId] = CursorDot(sessionId: sessionId, name: name, x: x, y: y));
        });
      } on ApiException catch (e) {
        fatal = e.status == 401 || e.status == 403 || e.status == 404;
      } catch (_) {}
      if (fatal || !mounted) return;
      await _backoff(backoff);
      backoff = backoff * 2 > const Duration(seconds: 5) ? const Duration(seconds: 5) : backoff * 2;
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
    _progressFuture = context.read<Session>().api.roadmapProgress(widget.projectId);
    _presenceLoop(context.read<Session>().api);
  }

  void _scrollToBand(int i) {
    if (!_canvasController.hasClients) return;
    final target = i * (RoadmapCanvas.bandWidth + RoadmapCanvas.bandSpacing) - 24;
    _canvasController.animateTo(
      target.clamp(0.0, _canvasController.position.maxScrollExtent).toDouble(),
      duration: const Duration(milliseconds: 300),
      curve: Curves.easeOutCubic,
    );
  }

  Future<void> _reload() async {
    final nodes = context.read<Session>().api.listRoadmap(widget.projectId);
    final progress = context.read<Session>().api.roadmapProgress(widget.projectId);
    setState(() {
      _future = nodes;
      _progressFuture = progress;
    });
    await Future.wait([nodes, progress]);
  }

  Future<void> _addRoot() async {
    final l10n = context.read<Appearance>();
    final title = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.t('roadmap.addRoot')),
        content: TextField(controller: title, decoration: InputDecoration(labelText: l10n.t('roadmap.rootPlaceholder'))),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('roadmap.add'))),
        ],
      ),
    );
    if (ok == true && title.text.trim().isNotEmpty && mounted) {
      await context.read<Session>().api.createRoadmapNode(widget.projectId, title.text.trim());
      await _reload();
    }
    title.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    return AppScaffold(
      actions: [
        SegmentedButton<_RoadmapView>(
          segments: [
            ButtonSegment(value: _RoadmapView.progress, label: Text(l10n.t('roadmap.viewProgress'))),
            ButtonSegment(value: _RoadmapView.bands, label: Text(l10n.t('roadmap.viewBands'))),
            ButtonSegment(value: _RoadmapView.canvas, label: Text(l10n.t('roadmap.viewCanvas'))),
          ],
          selected: {_view},
          onSelectionChanged: (s) => setState(() => _view = s.first),
        ),
      ],
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(28, 64, 28, 8),
            child: Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(l10n.t('roadmap.title').toUpperCase(), style: theme.textTheme.displaySmall),
                      const SizedBox(height: 8),
                      Text(l10n.t('roadmap.lead'), style: theme.textTheme.bodyLarge),
                    ],
                  ),
                ),
                FilledButton(onPressed: _addRoot, child: Text(l10n.t('roadmap.newItem'))),
              ],
            ),
          ),
          if (_presence.isNotEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 6),
              child: Wrap(
                spacing: 6,
                children: [
                  Icon(Icons.groups_2_outlined, size: 16, color: Cobalt.of(context).brand),
                  for (final e in _presence) Chip(label: Text(e.name, style: theme.textTheme.bodySmall)),
                ],
              ),
            ),
          Expanded(
            child: _view == _RoadmapView.progress
                ? FutureBuilder(
                    future: _progressFuture,
                    builder: (context, snap) {
                      if (!snap.hasData) {
                        return Center(child: snap.hasError ? Text('${snap.error}') : const CircularProgressIndicator());
                      }
                      return RoadmapProgressView(progress: snap.data!, emptyLabel: l10n.t('roadmap.empty'));
                    },
                  )
                : FutureBuilder(
                    future: _future,
                    builder: (context, snap) {
                      if (!snap.hasData) {
                        return Center(child: snap.hasError ? Text('${snap.error}') : const CircularProgressIndicator());
                      }
                      if (snap.data!.isEmpty) return Center(child: Text(l10n.t('roadmap.empty')));
                      if (_view == _RoadmapView.bands) {
                        return RoadmapBands(projectId: widget.projectId, nodes: snap.data!, onReload: _reload);
                      }
                      return Column(
                        children: [
                          RoadmapMinimap(nodes: snap.data!, onTapBand: _scrollToBand),
                          const Divider(height: 1),
                          Expanded(
                            child: RoadmapCanvas(
                              nodes: snap.data!,
                              tileBuilder: (node, _) => RoadmapNodeCard(projectId: widget.projectId, node: node, onReload: _reload),
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
