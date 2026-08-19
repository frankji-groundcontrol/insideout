import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/errors.dart';
import '../../api/models.dart';
import '../../app.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import 'coach_panel.dart';

class PrdPage extends StatefulWidget {
  const PrdPage({super.key, required this.id});

  final String id;

  @override
  State<PrdPage> createState() => _PrdPageState();
}

class _PrdPageState extends State<PrdPage> {
  Prd? prd;
  Conversation? conversation;
  List<ChatMessage> messages = [];
  String? error;
  bool loading = true;
  final titleCtrl = TextEditingController();
  final sectionCtrls = <String, TextEditingController>{};

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    titleCtrl.dispose();
    for (final c in sectionCtrls.values) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _load() async {
    final api = context.read<Session>().api;
    try {
      final loaded = await api.getPrd(widget.id);
      final conv = await api.conversationForPrd(widget.id);
      var msgs = <ChatMessage>[];
      if (conv != null) msgs = await api.listMessages(conv.id);
      titleCtrl.text = loaded.title;
      for (final key in prdSectionKeys) {
        sectionCtrls[key]?.dispose();
        sectionCtrls[key] = TextEditingController(text: loaded.sections[key] ?? '');
      }
      setState(() {
        prd = loaded;
        conversation = conv;
        messages = msgs;
        loading = false;
      });
    } catch (e) {
      setState(() {
        error = e.toString();
        loading = false;
      });
    }
  }

  Future<void> _save({bool titleOnly = false}) async {
    final sections = titleOnly ? <String, String>{} : {for (final e in sectionCtrls.entries) e.key: e.value.text};
    final updated = await context.read<Session>().api.updatePrd(
          widget.id,
          title: titleCtrl.text.trim(),
          sections: titleOnly ? null : sections,
        );
    setState(() => prd = updated);
  }

  Future<void> _snapshot() async {
    final note = TextEditingController();
    final l10n = context.read<Appearance>();
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
      await context.read<Session>().api.createRevision(widget.id, note: note.text.trim().isEmpty ? null : note.text.trim());
    }
    note.dispose();
  }

  Future<void> _build() async {
    final api = context.read<Session>().api;
    final l10n = context.read<Appearance>();
    try {
      final result = await api.buildFromPrd(widget.id);
      if (mounted) _goProject(result);
    } on ApiException catch (e) {
      if (e.code == 'replace_conflict' && e.liveCount != null) {
        final ok = await showDialog<bool>(
          context: context,
          builder: (ctx) => AlertDialog(
            title: Text(l10n.t('prd.buildMVP')),
            content: Text(l10n.t('prd.buildReplaceConfirm', {'count': e.liveCount})),
            actions: [
              TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
              FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('common.confirm'))),
            ],
          ),
        );
        if (ok == true && mounted) {
          final result = await api.buildFromPrd(widget.id, expectedCount: e.liveCount);
          if (mounted) _goProject(result);
        }
      } else {
        rethrow;
      }
    }
  }

  void _goProject(Map<String, dynamic> result) {
    final projectId = result['projectId']?.toString();
    if (projectId != null && projectId.isNotEmpty) {
      context.go('/projects/$projectId');
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return AppScaffold(
      title: prd?.title ?? l10n.t('prd.title'),
      body: loading
          ? const Center(child: CircularProgressIndicator())
          : error != null && prd == null
              ? Center(child: Text(error!))
              : Row(
                  children: [
                    Expanded(
                      flex: 3,
                      child: ListView(
                        padding: const EdgeInsets.all(16),
                        children: [
                          TextField(
                            controller: titleCtrl,
                            decoration: InputDecoration(labelText: l10n.t('prd.title')),
                            onSubmitted: (_) => _save(titleOnly: true),
                          ),
                          Wrap(
                            spacing: 8,
                            children: [
                              TextButton(onPressed: () => context.go('/prd/${widget.id}/revisions'), child: Text(l10n.t('prd.revisionHistory'))),
                              TextButton(onPressed: () => context.go('/prd/${widget.id}/export'), child: Text(l10n.t('prd.exportMarkdown'))),
                              TextButton(onPressed: _snapshot, child: Text(l10n.t('prd.snapshotRevision'))),
                              DropdownButton<String>(
                                value: prd?.status,
                                hint: Text(l10n.t('project.statusLabel')),
                                items: const ['draft', 'reviewing', 'approved', 'rejected']
                                    .map((s) => DropdownMenuItem(value: s, child: Text(s)))
                                    .toList(),
                                onChanged: (status) async {
                                  if (status == null) return;
                                  final updated = await context.read<Session>().api.updatePrdStatus(widget.id, status);
                                  setState(() => prd = updated);
                                },
                              ),
                              OutlinedButton(onPressed: _build, child: Text(l10n.t('prd.buildMVP'))),
                              FilledButton(onPressed: _save, child: Text(l10n.t('common.save'))),
                            ],
                          ),
                          ...prdSectionKeys.map(
                            (key) => Padding(
                              padding: const EdgeInsets.only(bottom: 16),
                              child: TextField(
                                controller: sectionCtrls[key],
                                maxLines: 4,
                                decoration: InputDecoration(labelText: l10n.t('prd.sections.$key'), border: const OutlineInputBorder()),
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                    const VerticalDivider(width: 1),
                    Expanded(
                      flex: 2,
                      child: CoachPanel(
                        prdId: widget.id,
                        conversation: conversation,
                        messages: messages,
                        onPrdUpdated: () async {
                          final fresh = await context.read<Session>().api.getPrd(widget.id);
                          if (!mounted) return;
                          for (final key in prdSectionKeys) {
                            sectionCtrls[key]?.text = fresh.sections[key] ?? '';
                          }
                          setState(() => prd = fresh);
                        },
                        onStageChanged: (stage) {
                          final convNow = conversation;
                          if (convNow == null) return;
                          setState(() {
                            conversation = Conversation(id: convNow.id, prdId: convNow.prdId, stage: stage, status: convNow.status);
                          });
                        },
                      ),
                    ),
                  ],
                ),
    );
  }
}
