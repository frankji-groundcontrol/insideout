import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/errors_l10n.dart';
import '../../api/errors.dart';
import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/loop_focus.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import 'coach_panel.dart';
import 'prd_desk.dart';

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
    context.read<LoopFocus>().rememberPrd(widget.id);
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
      context.read<LoopFocus>().rememberWorkspace(loaded.workspaceId);
      if (loaded.projectId != null) {
        context.read<LoopFocus>().rememberProject(loaded.projectId!, workspaceId: loaded.workspaceId);
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

  Future<void> _requestReview() async {
    final updated = await context.read<Session>().api.updatePrdStatus(widget.id, 'reviewing');
    setState(() => prd = updated);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final wide = MediaQuery.sizeOf(context).width >= 900;
    final desk = loading
        ? const Center(child: CircularProgressIndicator())
        : error != null && prd == null
            ? Center(child: Text(localizedError(error!, l10n.t)))
            : (wide
                ? Row(
                    children: [
                      Expanded(flex: 3, child: _prdColumn()),
                      Expanded(flex: 2, child: _coach()),
                    ],
                  )
                : ListView(
                    children: [
                      SizedBox(height: 640, child: _prdColumn()),
                      SizedBox(height: 520, child: _coach()),
                    ],
                  ));
    return AppScaffold(body: desk);
  }

  Widget _coach() {
    return CoachPanel(
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
    );
  }

  Widget _prdColumn() {
    return PrdDeskColumn(
      prdId: widget.id,
      prd: prd,
      titleCtrl: titleCtrl,
      sectionCtrls: sectionCtrls,
      onSave: _save,
      onSaveTitle: () => _save(titleOnly: true),
      onRequestReview: _requestReview,
      onSnapshot: _snapshot,
      onBuild: _build,
      onStatus: (status) async {
        final updated = await context.read<Session>().api.updatePrdStatus(widget.id, status);
        if (mounted) setState(() => prd = updated);
      },
    );
  }
}
