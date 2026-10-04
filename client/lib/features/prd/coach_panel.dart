import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/errors_l10n.dart';
import '../../api/errors.dart';
import '../../api/models.dart';
import '../../api/sse.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_theme.dart';

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
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text(context.read<Appearance>().t('coach.factRecorded', {'fact': fact}))),
                );
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
    final palette = Cobalt.of(context);
    final stage = widget.conversation?.stage ?? 'clarify';
    final suggestions = l10n.suggestions(stage);
    final ink = palette.brightnessIsDark ? palette.fgPrimary : Cobalt.darkType;
    final field = palette.brightnessIsDark ? palette.surfaceSunken : Cobalt.hue950;
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
    return ColoredBox(
      color: field,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(28, 64, 16, 8),
            child: Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        l10n.t('coach.title').toUpperCase(),
                        style: cobaltSerif(palette, size: 32, color: ink, weight: FontWeight.w700),
                      ),
                      const SizedBox(height: 6),
                      Text(
                        l10n.t('coach.pullQuotes'),
                        style: Theme.of(context).textTheme.bodySmall?.copyWith(color: ink.withValues(alpha: 0.7)),
                      ),
                      Container(width: 36, height: 3, margin: const EdgeInsets.only(top: 10), color: Cobalt.hue500),
                    ],
                  ),
                ),
                IconButton(
                  tooltip: l10n.t('coach.closeCoach'),
                  onPressed: () => setState(() => open = false),
                  color: ink,
                  icon: const Icon(Icons.close),
                ),
              ],
            ),
          ),
          if (widget.conversation == null)
            Padding(
              padding: const EdgeInsets.all(24),
              child: Text(l10n.t('coach.noConversation'), style: cobaltSerif(palette, size: 20, color: ink)),
            )
          else ...[
            if (suggestions.isNotEmpty)
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 20),
                child: Wrap(
                  spacing: 8,
                  children: [
                    for (final prompt in suggestions)
                      ActionChip(label: Text(prompt), onPressed: sending || retry > 0 ? null : () => _send(prompt)),
                  ],
                ),
              ),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(28, 16, 28, 16),
                children: [
                  ...widget.messages.map((m) => _Quote(text: m.content, role: m.role, ink: ink)),
                  if (stream.isNotEmpty) _Quote(text: stream, role: 'assistant', ink: ink),
                ],
              ),
            ),
          ],
          if (error != null) Padding(padding: const EdgeInsets.all(12), child: Text(localizedError(error!, l10n.t), style: TextStyle(color: ink))),
          if (retry > 0)
            Padding(
              padding: const EdgeInsets.all(12),
              child: Text(
                l10n.t(error != null && error!.contains('unavailable') ? 'coach.serviceUnavailable' : 'coach.rateLimited', {'seconds': retry}),
                style: TextStyle(color: ink),
              ),
            ),
          if (widget.conversation != null)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
              child: Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: draft,
                      style: cobaltSerif(palette, size: 17.5, color: ink, weight: FontWeight.w400),
                      enabled: !sending && retry == 0,
                      decoration: InputDecoration(
                        hintText: sending ? l10n.t('coach.thinking') : l10n.t('coach.placeholder'),
                        hintStyle: TextStyle(color: ink.withValues(alpha: 0.5)),
                        fillColor: field,
                      ),
                    ),
                  ),
                  IconButton(onPressed: sending || retry > 0 ? null : _send, color: ink, icon: const Icon(Icons.send)),
                ],
              ),
            ),
        ],
      ),
    );
  }
}

extension on Cobalt {
  bool get brightnessIsDark => surfaceBase == Cobalt.darkGround;
}

class _Quote extends StatelessWidget {
  const _Quote({required this.text, required this.role, required this.ink});

  final String text;
  final String role;
  final Color ink;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 28),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            '“$text”',
            style: cobaltSerif(palette, size: 26, color: ink, weight: FontWeight.w600, height: 1.35),
          ),
          const SizedBox(height: 8),
          Text(role, style: Theme.of(context).textTheme.labelSmall?.copyWith(color: ink.withValues(alpha: 0.6))),
        ],
      ),
    );
  }
}
