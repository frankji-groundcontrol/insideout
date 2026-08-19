import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';

class RevisionsPage extends StatefulWidget {
  const RevisionsPage({super.key, required this.prdId});

  final String prdId;

  @override
  State<RevisionsPage> createState() => _RevisionsPageState();
}

class _RevisionsPageState extends State<RevisionsPage> {
  late Future<List<PrdRevision>> _future;
  String? selectedId;

  @override
  void initState() {
    super.initState();
    _future = context.read<Session>().api.listRevisions(widget.prdId);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return AppScaffold(
      title: l10n.t('prd.revisionsTitle'),
      fab: FloatingActionButton(
        onPressed: () async {
          final note = TextEditingController();
          final ok = await showDialog<bool>(
            context: context,
            builder: (ctx) => AlertDialog(
              title: Text(l10n.t('prd.snapshotRevision')),
              content: TextField(controller: note, decoration: InputDecoration(labelText: l10n.t('prd.revisionNote'))),
              actions: [
                TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
                FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('common.save'))),
              ],
            ),
          );
          if (ok == true && mounted) {
            await context.read<Session>().api.createRevision(widget.prdId, note: note.text.trim().isEmpty ? null : note.text.trim());
            setState(() => _future = context.read<Session>().api.listRevisions(widget.prdId));
          }
          note.dispose();
        },
        child: const Icon(Icons.add),
      ),
      body: FutureBuilder(
        future: _future,
        builder: (context, snap) {
          if (!snap.hasData) return Center(child: snap.hasError ? Text('${snap.error}') : const CircularProgressIndicator());
          if (snap.data!.isEmpty) {
            return Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(l10n.t('prd.emptyRevisions')),
                  TextButton(onPressed: () => context.go('/prd/${widget.prdId}'), child: Text(l10n.t('prd.backToPrd'))),
                ],
              ),
            );
          }
          final selected = snap.data!.cast<PrdRevision?>().firstWhere((r) => r!.id == selectedId, orElse: () => snap.data!.first)!;
          return Row(
            children: [
              SizedBox(
                width: 260,
                child: ListView(
                  children: snap.data!
                      .map(
                        (r) => ListTile(
                          selected: r.id == selected.id,
                          title: Text(l10n.t('prd.revisionN', {'n': r.revision})),
                          subtitle: Text(r.note?.isNotEmpty == true ? r.note! : r.createdAt),
                          onTap: () => setState(() => selectedId = r.id),
                        ),
                      )
                      .toList(),
                ),
              ),
              const VerticalDivider(width: 1),
              Expanded(
                child: ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    for (final key in prdSectionKeys)
                      Padding(
                        padding: const EdgeInsets.only(bottom: 16),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(l10n.t('prd.sections.$key'), style: Theme.of(context).textTheme.titleSmall),
                            const SizedBox(height: 4),
                            Text(selected.sections[key]?.isNotEmpty == true ? selected.sections[key]! : '—'),
                          ],
                        ),
                      ),
                  ],
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}
