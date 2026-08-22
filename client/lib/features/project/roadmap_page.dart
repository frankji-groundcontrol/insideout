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
  bool _canvas = false;
  final _canvasController = ScrollController();

  @override
  void dispose() {
    _canvasController.dispose();
    super.dispose();
  }

  @override
  void initState() {
    super.initState();
    _future = context.read<Session>().api.listRoadmap(widget.projectId);
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
    setState(() => _future = context.read<Session>().api.listRoadmap(widget.projectId));
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
                          await api.updateRoadmapNode(n.id, title: title.text.trim(), description: desc.text, status: status);
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
      body: FutureBuilder(
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
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}
