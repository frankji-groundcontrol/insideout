import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/errors.dart';
import '../../api/models.dart';
import '../../api/sse.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';

class CoachPanel extends StatefulWidget {
  const CoachPanel({
    super.key,
    required this.prdId,
    required this.conversation,
    required this.messages,
    required this.onPrdUpdated,
    required this.onStageChanged,
  });

  final String prdId;
  final Conversation? conversation;
  final List<ChatMessage> messages;
  final Future<void> Function() onPrdUpdated;
  final void Function(String stage) onStageChanged;

  @override
  State<CoachPanel> createState() => _CoachPanelState();
}

class _CoachPanelState extends State<CoachPanel> {
  final draft = TextEditingController();
  String stream = '';
  String? error;
  bool sending = false;
  bool open = true;
  int retry = 0;
  Timer? retryTimer;

  @override
  void dispose() {
    retryTimer?.cancel();
    draft.dispose();
    super.dispose();
  }

  void _startRetry(int seconds) {
    retryTimer?.cancel();
    setState(() => retry = seconds);
    retryTimer = Timer.periodic(const Duration(seconds: 1), (timer) {
      if (retry <= 1) {
        timer.cancel();
        setState(() => retry = 0);
      } else {
        setState(() => retry -= 1);
      }
    });
  }

  Future<void> _send([String? preset]) async {
    final conv = widget.conversation;
    final text = (preset ?? draft.text).trim();
    if (conv == null || text.isEmpty || sending || retry > 0) return;
    setState(() {
      sending = true;
      stream = '';
      error = null;
    });
    draft.clear();
    try {
      await context.read<Session>().api.sendCoach(
            conv.id,
            text,
            CoachHandlers(
              onDelta: (chunk) => setState(() => stream += chunk),
              onPrdUpdated: (_) => widget.onPrdUpdated(),
              onStageChanged: widget.onStageChanged,
              onFactRecorded: (fact) {
                if (!mounted) return;
                ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Fact: $fact')));
              },
            ),
          );
    } on ApiException catch (e) {
      setState(() => error = e.message);
      if (isCoachBackoff(e)) _startRetry(coachRetrySeconds(e));
    } catch (e) {
      setState(() => error = e.toString());
    } finally {
      if (mounted) setState(() => sending = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final stage = widget.conversation?.stage ?? 'clarify';
    final suggestions = l10n.suggestions(stage);
    if (!open) {
      return Align(
        alignment: Alignment.centerRight,
        child: IconButton(
          tooltip: l10n.t('coach.openCoach'),
          onPressed: () => setState(() => open = true),
          icon: const Icon(Icons.chat_bubble_outline),
        ),
      );
    }
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.all(8),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  '${l10n.t('coach.title')} · ${l10n.t('coach.stage.$stage')}',
                  style: Theme.of(context).textTheme.titleSmall,
                ),
              ),
              IconButton(
                tooltip: l10n.t('coach.closeCoach'),
                onPressed: () => setState(() => open = false),
                icon: const Icon(Icons.close),
              ),
            ],
          ),
        ),
        if (widget.conversation == null)
          Padding(
            padding: const EdgeInsets.all(16),
            child: Text(l10n.t('coach.noConversation')),
          )
        else ...[
          if (suggestions.isNotEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 8),
              child: Wrap(
                spacing: 8,
                children: [
                  Text(l10n.t('coach.suggestTitle'), style: Theme.of(context).textTheme.labelMedium),
                  for (final prompt in suggestions)
                    ActionChip(label: Text(prompt), onPressed: sending || retry > 0 ? null : () => _send(prompt)),
                ],
              ),
            ),
          Expanded(
            child: ListView(
              padding: const EdgeInsets.all(8),
              children: [
                ...widget.messages.map((m) => ListTile(dense: true, title: Text(m.role), subtitle: Text(m.content))),
                if (stream.isNotEmpty) ListTile(dense: true, title: const Text('assistant'), subtitle: Text(stream)),
              ],
            ),
          ),
        ],
        if (error != null) Padding(padding: const EdgeInsets.all(8), child: Text(error!)),
        if (retry > 0)
          Padding(
            padding: const EdgeInsets.all(8),
            child: Text(l10n.t(error != null && error!.contains('unavailable') ? 'coach.serviceUnavailable' : 'coach.rateLimited', {'seconds': retry})),
          ),
        if (widget.conversation != null)
          Padding(
            padding: const EdgeInsets.all(8),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: draft,
                    enabled: !sending && retry == 0,
                    decoration: InputDecoration(hintText: sending ? l10n.t('coach.thinking') : l10n.t('coach.placeholder')),
                  ),
                ),
                IconButton(onPressed: sending || retry > 0 ? null : _send, icon: const Icon(Icons.send)),
              ],
            ),
          ),
      ],
    );
  }
}
