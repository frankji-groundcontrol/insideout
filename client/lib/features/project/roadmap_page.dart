import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/seal_chip.dart';
import 'roadmap_canvas.dart';

class RoadmapPage extends StatefulWidget {
  const RoadmapPage({super.key, required this.projectId});

  final String projectId;

  @override
  State<RoadmapPage> createState() => _RoadmapPageState();
}

class _RoadmapPageState extends State<RoadmapPage> {
  late Future<List<RoadmapNode>> _future;
  RoadmapProgress? _progress;
  bool _canvas = false;
  final _canvasController = ScrollController();
  List<PresenceEntry> _presence = [];
  final Map<String, CursorDot> _cursors = {};
  DateTime _lastCursorSent = DateTime.fromMillisecondsSinceEpoch(0);
  late final String _clientId = DateTime.now().microsecondsSinceEpoch.toRadixString(36);

  void _sendCursor(double x, double y) {
    final now = DateTime.now();
    if (now.difference(_lastCursorSent).inMilliseconds < 200) return;
    _lastCursorSent = now;
    context.read<Session>().api.cursor(widget.projectId, x, y).catchError((_) {});
  }

  void _watchPresence() {
    context.read<Session>().api.presenceStream(widget.projectId, _clientId, (entries) {
      if (!mounted) return;
      final live = entries.map((e) => e.sessionId).toSet();
      _cursors.removeWhere((id, _) => !live.contains(id));
      setState(() => _presence = entries);
    }, onCursor: (sessionId, name, x, y) {
      if (!mounted || sessionId == _clientId) return;
      setState(() => _cursors[sessionId] = CursorDot(sessionId: sessionId, name: name, x: x, y: y));
    }).catchError((_) {});
  }

  @override
  void dispose() {
    _canvasController.dispose();
    super.dispose();
  }

  @override
  void initState() {
    super.initState();
    _future = context.read<Session>().api.listRoadmap(widget.projectId);
    _watchPresence();
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
    final api = context.read<Session>().api;
    setState(() => _future = api.listRoadmap(widget.projectId));
    try {
      final p = await api.roadmapProgress(widget.projectId);
      if (mounted) setState(() => _progress = p);
    } catch (_) {}
  }

  List<RoadmapNode> _children(List<RoadmapNode> all, String? parentId) {
    return all.where((n) => n.parentId == parentId).toList()..sort((a, b) => a.position.compareTo(b.position));
  }

  Widget _tree(List<RoadmapNode> all, String? parentId, int depth) {
    return Column(
      children: _children(all, parentId)
          .map(
            (n) => Column(
              children: [
                _nodeTile(all, n, 16.0 + depth * 20),
                _tree(all, n.id, depth + 1),
              ],
            ),
          )
          .toList(),
    );
  }

  /// One node row with its seal mark, status chip, attribution, and the
  /// edit/add/move/expand/delete menu — shared by list and canvas.
  Widget _progressStripItem(String label, List<String> items) {
    final theme = Theme.of(context);
    return Expanded(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(label, style: theme.textTheme.titleSmall),
          if (items.isEmpty) Text('—', style: theme.textTheme.bodySmall),
          for (final t in items.take(3))
            Text(t, style: theme.textTheme.bodySmall, overflow: TextOverflow.ellipsis),
        ],
      ),
    );
  }

  Widget _nodeTile(List<RoadmapNode> all, RoadmapNode n, double left) {
    final l10n = context.watch<Appearance>();
    return ListTile(
                  contentPadding: EdgeInsets.only(left: left, right: 12),
                  leading: SealMark(status: n.status),
                  title: Text(n.title),
                  subtitle: Wrap(
                    crossAxisAlignment: WrapCrossAlignment.center,
                    spacing: 8,
                    children: [
                      SealChip(label: l10n.t('roadmap.status.${n.status}'), tone: SealChip.fromStatus(n.status)),
                      if (n.deadline != null)
                        SealChip(
                          label: '${n.deadline!.substring(0, 10)}${n.pressure != null && n.pressure != 'normal' ? ' · ${l10n.t('roadmap.pressure.${n.pressure}')}' : ''}',
                          tone: n.pressure == 'overdue' || n.pressure == 'high_risk' ? SealTone.danger : SealTone.neutral,
                        ),
                      if (n.editorName != null) Text(n.editorName!, style: Theme.of(context).textTheme.bodySmall),
                    ],
                  ),
                  trailing: PopupMenuButton<String>(
                    onSelected: (value) async {
                      final api = context.read<Session>().api;
                      if (value == 'edit') {
                        final title = TextEditingController(text: n.title);
                        final desc = TextEditingController(text: n.description);
                        var status = n.status;
                        String? deadline = n.deadline; // "" clears; null untouched
                        final ok = await showDialog<bool>(
                          context: context,
                          builder: (ctx) => StatefulBuilder(
                            builder: (ctx, setLocal) => AlertDialog(
                              title: Text(l10n.t('roadmap.edit')),
                              content: Column(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  TextField(controller: title, decoration: InputDecoration(labelText: l10n.t('roadmap.rootPlaceholder'))),
                                  TextField(controller: desc, decoration: InputDecoration(labelText: l10n.t('roadmap.descPlaceholder'))),
                                  DropdownButton<String>(
                                    value: status,
                                    items: ['locked', 'pending', 'in_progress', 'done']
                                        .map((s) => DropdownMenuItem(value: s, child: Text(l10n.t('roadmap.status.$s'))))
                                        .toList(),
                                    onChanged: (v) => setLocal(() => status = v ?? status),
                                  ),
                                  StatefulBuilder(
                                    builder: (ctx, setDl) => Row(
                                      children: [
                                        TextButton(
                                          onPressed: () async {
                                            final picked = await showDatePicker(
                                              context: ctx,
                                              initialDate: DateTime.now().add(const Duration(days: 7)),
                                              firstDate: DateTime.now().subtract(const Duration(days: 1)),
                                              lastDate: DateTime.now().add(const Duration(days: 365 * 3)),
                                            );
                                            if (picked != null) setDl(() => deadline = picked.toUtc().toIso8601String());
                                          },
                                          child: Text(deadline == null ? l10n.t('roadmap.setDeadline') : deadline!.substring(0, 10)),
                                        ),
                                        if (deadline != null)
                                          IconButton(
                                            tooltip: l10n.t('roadmap.clearDeadline'),
                                            onPressed: () => setDl(() => deadline = ''),
                                            icon: const Icon(Icons.event_busy, size: 18),
                                          ),
                                      ],
                                    ),
                                  ),
                                ],
                              ),
                              actions: [
                                TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
                                FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('roadmap.save'))),
                              ],
                            ),
                          ),
                        );
                        if (ok == true && mounted) {
                          await api.updateRoadmapNode(n.id, title: title.text.trim(), description: desc.text, status: status,
                              deadline: deadline == '' ? '' : deadline);
                          await _reload();
                        }
                        title.dispose();
                        desc.dispose();
                      } else if (value == 'child') {
                        final title = TextEditingController();
                        final ok = await showDialog<bool>(
                          context: context,
                          builder: (ctx) => AlertDialog(
                            title: Text(l10n.t('roadmap.addChild')),
                            content: TextField(controller: title, decoration: InputDecoration(labelText: l10n.t('roadmap.childPlaceholder'))),
                            actions: [
                              TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
                              FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('roadmap.add'))),
                            ],
                          ),
                        );
                        if (ok == true && title.text.trim().isNotEmpty && mounted) {
                          await api.createRoadmapNode(widget.projectId, title.text.trim(), parentId: n.id);
                          await _reload();
                        }
                        title.dispose();
                      } else if (value == 'up') {
                        await api.moveRoadmapNode(n.id, parentId: n.parentId, position: n.position == 0 ? 0 : n.position - 1);
                        await _reload();
                      } else if (value == 'down') {
                        await api.moveRoadmapNode(n.id, parentId: n.parentId, position: n.position + 1);
                        await _reload();
                      } else if (value == 'expand') {
                        await api.expandRoadmapNode(n.id);
                        await _reload();
                      } else if (value == 'delete') {
                        await api.deleteRoadmapNode(n.id);
                        await _reload();
                      }
                    },
                    itemBuilder: (_) {
                      final l10n = context.read<Appearance>();
                      return [
                        PopupMenuItem(value: 'edit', child: Text(l10n.t('roadmap.edit'))),
                        PopupMenuItem(value: 'child', child: Text(l10n.t('roadmap.addChild'))),
                        PopupMenuItem(value: 'up', child: Text(l10n.t('roadmap.moveUp'))),
                        PopupMenuItem(value: 'down', child: Text(l10n.t('roadmap.moveDown'))),
                        PopupMenuItem(value: 'expand', child: Text(l10n.t('roadmap.expandAI'))),
                        PopupMenuItem(value: 'delete', child: Text(l10n.t('roadmap.delete'))),
                      ];
                    },
                  ),
    );
  }


  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return AppScaffold(
      title: l10n.t('roadmap.title'),
      actions: [
        SegmentedButton<bool>(
          segments: [
            ButtonSegment(value: false, label: Text(l10n.t('roadmap.viewList')), icon: const Icon(Icons.list)),
            ButtonSegment(value: true, label: Text(l10n.t('roadmap.viewCanvas')), icon: const Icon(Icons.view_column_outlined)),
          ],
          selected: {_canvas},
          onSelectionChanged: (s) => setState(() => _canvas = s.first),
        ),
        const SizedBox(width: 12),
      ],
      fab: FloatingActionButton(
        onPressed: () async {
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
        },
        child: const Icon(Icons.add),
      ),
      body: Column(
        children: [
          if (_progress != null)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
              child: Row(
                children: [
                  _progressStripItem(l10n.t('roadmap.progressNow'),
                      [..._progress!.now.map((i) => '${i.title}${i.pressure == 'overdue' || i.pressure == 'high_risk' ? ' !' : ''}'),
                       ..._progress!.needsDeadline.map((i) => '${i.title} (${l10n.t('roadmap.needsDeadline')})')]),
                  const SizedBox(width: 12),
                  _progressStripItem(l10n.t('roadmap.progressNext'), _progress!.next.map((i) => i.title).toList()),
                  const SizedBox(width: 12),
                  Text('${l10n.t('roadmap.progressDone')}: ${_progress!.doneCount}', style: Theme.of(context).textTheme.bodySmall),
                ],
              ),
            ),
          if (_presence.isNotEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
              child: Row(
                children: [
                  Icon(Icons.groups_2_outlined, size: 16, color: Theme.of(context).colorScheme.secondary),
                  const SizedBox(width: 6),
                  Expanded(
                    child: Wrap(
                      spacing: 6,
                      children: [
                        for (final e in _presence)
                          Chip(
                            label: Text(e.name, style: Theme.of(context).textTheme.bodySmall),
                            visualDensity: VisualDensity.compact,
                          ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          Expanded(child: FutureBuilder(
        future: _future,
        builder: (context, snap) {
          if (!snap.hasData) return Center(child: snap.hasError ? Text('${snap.error}') : const CircularProgressIndicator());
          if (snap.data!.isEmpty) return Center(child: Text(l10n.t('roadmap.empty')));
          if (!_canvas) return ListView(children: [_tree(snap.data!, null, 0)]);
          return Column(
            children: [
              RoadmapMinimap(
                nodes: snap.data!,
                onTapBand: (i) => _scrollToBand(i),
              ),
              Divider(height: 1, color: Theme.of(context).colorScheme.outlineVariant),
              Expanded(
                child: RoadmapCanvas(
                  nodes: snap.data!,
                  tileBuilder: (node, indent) => _nodeTile(snap.data!, node, indent),
                  controller: _canvasController,
                  onCursor: _sendCursor,
                  cursors: _cursors.values.toList(),
                ),
              ),
            ],
          );
        },
          )),
        ],
      ),
    );
  }
}
