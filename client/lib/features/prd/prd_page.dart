import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/errors.dart';
import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/loop_focus.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_field.dart';
import '../../theme/block_modal.dart';
import '../../theme/block_state.dart';
import '../../theme/cobalt.dart';
import '../../widgets/guarded_action.dart';
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
  Object? error;
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
    final focus = context.read<LoopFocus>();
    try {
      final loaded = await api.getPrd(widget.id);
      final conv = await api.conversationForPrd(widget.id);
      var msgs = <ChatMessage>[];
      if (conv != null) msgs = await api.listMessages(conv.id);
      titleCtrl.text = loaded.title;
      for (final key in prdSectionKeys) {
        sectionCtrls[key]?.dispose();
        sectionCtrls[key] = TextEditingController(
          text: loaded.sections[key] ?? '',
        );
      }
      focus.rememberWorkspace(loaded.workspaceId);
      if (loaded.projectId != null) {
        focus.rememberProject(
          loaded.projectId!,
          workspaceId: loaded.workspaceId,
        );
      }
      if (!mounted) return;
      setState(() {
        prd = loaded;
        conversation = conv;
        messages = msgs;
        error = null;
        loading = false;
      });
    } catch (e) {
      // Keep the object, not its toString: localizedError maps ApiException
      // by status and a stringified error always falls through to generic.
      if (!mounted) return;
      setState(() {
        error = e;
        loading = false;
      });
    }
  }

  Future<void> _save({bool titleOnly = false}) async {
    final sections = titleOnly
        ? <String, String>{}
        : {for (final e in sectionCtrls.entries) e.key: e.value.text};
    await guardWrite(context, () async {
      final updated = await context.read<Session>().api.updatePrd(
        widget.id,
        title: titleCtrl.text.trim(),
        sections: titleOnly ? null : sections,
      );
      if (mounted) setState(() => prd = updated);
    });
  }

  Future<void> _snapshot() async {
    final note = TextEditingController();
    final t = context.read<Appearance>().t;
    final navigator = Navigator.of(context);
    final ok = await showBlockModal<bool>(
      context: context,
      title: t('prd.snapshotRevision'),
      body: BlockField(label: t('prd.revisionNote'), controller: note),
      footer: [
        BlockButton(
          label: t('common.cancel'),
          variant: BlockButtonVariant.secondary,
          onPressed: () => navigator.pop(false),
        ),
        BlockButton(
          label: t('common.save'),
          onPressed: () => navigator.pop(true),
        ),
      ],
    );
    if (ok == true && mounted) {
      await guardWrite(
        context,
        () => context.read<Session>().api.createRevision(
          widget.id,
          note: note.text.trim().isEmpty ? null : note.text.trim(),
        ),
      );
    }
    // Not disposed: the pop animation keeps the modal's field mounted past
    // this await, and the guard's SnackBar rebuilds the overlay — a disposed
    // controller would throw. GC'd once the modal unmounts.
  }

  Future<void> _build() async {
    final api = context.read<Session>().api;
    final t = context.read<Appearance>().t;
    // The replace-conflict retry is part of the write; anything else the
    // server rejects surfaces through the guard's SnackBar.
    await guardWrite(context, () async {
      try {
        final result = await api.buildFromPrd(widget.id);
        if (mounted) _goProject(result);
      } on ApiException catch (e) {
        if (e.code == 'replace_conflict' && e.liveCount != null) {
          if (!mounted) return;
          final ok = await showBlockConfirm(
            context: context,
            title: t('prd.buildMVP'),
            message: t('prd.buildReplaceConfirm', {'count': e.liveCount}),
            confirmText: t('common.confirm'),
            danger: true,
          );
          if (ok && mounted) {
            final result = await api.buildFromPrd(
              widget.id,
              expectedCount: e.liveCount,
            );
            if (mounted) _goProject(result);
          }
        } else {
          rethrow;
        }
      }
    });
  }

  void _goProject(Map<String, dynamic> result) {
    final projectId = result['projectId']?.toString();
    if (projectId != null && projectId.isNotEmpty) {
      context.go('/projects/$projectId/roadmap');
    }
  }

  Future<void> _requestReview() async {
    await guardWrite(context, () async {
      final updated = await context.read<Session>().api.updatePrdStatus(
        widget.id,
        'reviewing',
      );
      if (mounted) setState(() => prd = updated);
    });
  }

  @override
  Widget build(BuildContext context) {
    final wide = MediaQuery.sizeOf(context).width >= Cobalt.bpWide;
    final desk = loading
        ? const BlockLoading()
        : error != null && prd == null
        ? BlockError(error: error, onRetry: _load)
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
                    // 640, not 520: the coach chrome (header + prompt blocks +
                    // composer + dock clearance) leaves the quote list a real
                    // viewport instead of a negative one.
                    SizedBox(height: 640, child: _coach()),
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
          conversation = Conversation(
            id: convNow.id,
            prdId: convNow.prdId,
            stage: stage,
            status: convNow.status,
          );
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
        await guardWrite(context, () async {
          final updated = await context.read<Session>().api.updatePrdStatus(
            widget.id,
            status,
          );
          if (mounted) setState(() => prd = updated);
        });
      },
    );
  }
}
