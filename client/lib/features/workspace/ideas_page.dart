import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';

class IdeasPage extends StatefulWidget {
  const IdeasPage({super.key, required this.workspaceId});

  final String workspaceId;

  @override
  State<IdeasPage> createState() => _IdeasPageState();
}

class _IdeasPageState extends State<IdeasPage> {
  late Future<List<Idea>> _future;

  @override
  void initState() {
    super.initState();
    _future = context.read<Session>().api.listIdeas(widget.workspaceId);
  }

  Future<void> _reload() async {
    setState(() => _future = context.read<Session>().api.listIdeas(widget.workspaceId));
  }

  Future<void> _create() async {
    final l10n = context.read<Appearance>();
    final title = TextEditingController();
    final content = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.t('idea.capture')),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(controller: title, decoration: InputDecoration(labelText: l10n.t('idea.titlePlaceholder'))),
            TextField(controller: content, decoration: InputDecoration(labelText: l10n.t('idea.contentPlaceholder')), maxLines: 4),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('idea.capture'))),
        ],
      ),
    );
    if (ok == true && title.text.trim().isNotEmpty && mounted) {
      await context.read<Session>().api.createIdea(widget.workspaceId, title.text.trim(), content.text);
      await _reload();
    }
    title.dispose();
    content.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return AppScaffold(
      title: l10n.t('idea.title'),
      fab: FloatingActionButton(onPressed: _create, child: const Icon(Icons.add)),
      body: FutureBuilder(
        future: _future,
        builder: (context, snap) {
          if (!snap.hasData) return Center(child: snap.hasError ? Text('${snap.error}') : const CircularProgressIndicator());
          if (snap.data!.isEmpty) return Center(child: Text(l10n.t('idea.empty')));
          return ListView(
            children: snap.data!
                .map(
                  (idea) => ListTile(
                    title: Text(idea.title),
                    subtitle: Text(idea.status),
                    trailing: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        if (idea.prdId == null)
                          TextButton(
                            onPressed: () async {
                              final ids = await context.read<Session>().api.convertIdea(idea.id);
                              if (context.mounted) context.go('/prd/${ids['prdId']}');
                            },
                            child: Text(l10n.t('idea.convert')),
                          )
                        else
                          TextButton(
                            onPressed: () => context.go('/prd/${idea.prdId}'),
                            child: Text(l10n.t('prd.title')),
                          ),
                        PopupMenuButton<String>(
                          onSelected: (value) async {
                            final api = context.read<Session>().api;
                            if (value == 'edit') {
                              final title = TextEditingController(text: idea.title);
                              final content = TextEditingController(text: idea.content);
                              final ok = await showDialog<bool>(
                                context: context,
                                builder: (ctx) => AlertDialog(
                                  title: Text(l10n.t('common.edit')),
                                  content: Column(
                                    mainAxisSize: MainAxisSize.min,
                                    children: [
                                      TextField(controller: title, decoration: InputDecoration(labelText: l10n.t('idea.titlePlaceholder'))),
                                      TextField(controller: content, decoration: InputDecoration(labelText: l10n.t('idea.contentPlaceholder')), maxLines: 4),
                                    ],
                                  ),
                                  actions: [
                                    TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
                                    FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('common.save'))),
                                  ],
                                ),
                              );
                              if (ok == true && mounted) {
                                await api.updateIdea(idea.id, title: title.text.trim(), content: content.text);
                                await _reload();
                              }
                              title.dispose();
                              content.dispose();
                            } else if (value == 'drop') {
                              await api.dropIdea(idea.id);
                              await _reload();
                            }
                          },
                          itemBuilder: (_) => [
                            PopupMenuItem(value: 'edit', child: Text(l10n.t('common.edit'))),
                            PopupMenuItem(value: 'drop', child: Text(l10n.t('idea.drop'))),
                          ],
                        ),
                      ],
                    ),
                  ),
                )
                .toList(),
          );
        },
      ),
    );
  }
}
