import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/errors_l10n.dart';
import '../../api/errors.dart';
import '../../api/models.dart';
import '../../api/sse.dart';
import '../../chrome/product_dock.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_chrome.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_theme.dart';
import 'coach_column.dart';

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
  Object? error;
  bool sending = false;
  bool open = true;
  int retry = 0;
  Timer? retryTimer;
  List<Map<String, dynamic>> findings = const [];

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
              SnackBar(
                content: Text(
                  context.read<Appearance>().t('coach.factRecorded', {
                    'fact': fact,
                  }),
                ),
              ),
            );
          },
          onCriticFindings: (next) {
            if (!mounted) return;
            setState(() => findings = next);
          },
        ),
      );
    } on ApiException catch (e) {
      setState(() => error = e.message);
      if (isCoachBackoff(e)) _startRetry(coachRetrySeconds(e));
    } catch (e) {
      // Keep the object, not its toString: TimeoutException from the
      // stream's idle timeout maps to error.network in localizedError.
      if (mounted) setState(() => error = e);
    } finally {
      if (mounted) setState(() => sending = false);
    }
  }

  bool get _busy => sending || retry > 0;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final palette = Cobalt.of(context);
    final stage = widget.conversation?.stage ?? 'clarify';
    final err = error;
    final suggestions = l10n.suggestions(stage);
    final ink = coachInk(context);
    if (!open) {
      return Align(
        alignment: Alignment.centerRight,
        child: Padding(
          padding: const EdgeInsets.all(Cobalt.space4),
          child: BlockIconButton(
            icon: Icons.chat_bubble_outline,
            tooltip: l10n.t('coach.openCoach'),
            onPressed: () => setState(() => open = true),
          ),
        ),
      );
    }
    return ColoredBox(
      color: coachSlab(context),
      child: Padding(
        padding: EdgeInsets.only(
          bottom:
              ProductDock.height +
              Cobalt.space4 +
              MediaQuery.of(context).padding.bottom,
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _header(context, palette, l10n, stage, ink),
            if (widget.conversation == null)
              Padding(
                padding: const EdgeInsets.all(Cobalt.space6),
                child: Text(
                  l10n.t('coach.noConversation'),
                  style: cobaltSerif(palette, size: 20, color: ink),
                ),
              )
            else ...[
              if (suggestions.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: Cobalt.space6,
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        l10n.t('coach.suggestTitle'),
                        style: cobaltSerif(
                          palette,
                          size: 12,
                          weight: FontWeight.w600,
                          color: ink.withValues(alpha: 0.7),
                        ),
                      ),
                      const SizedBox(height: Cobalt.space2),
                      Wrap(
                        spacing: Cobalt.space2,
                        runSpacing: Cobalt.space2,
                        children: [
                          for (final prompt in suggestions)
                            CoachPrompt(
                              label: prompt,
                              onTap: _busy ? null : () => _send(prompt),
                            ),
                        ],
                      ),
                    ],
                  ),
                ),
              Expanded(
                child: ListView(
                  padding: const EdgeInsets.fromLTRB(
                    Cobalt.space6,
                    Cobalt.space4,
                    Cobalt.space6,
                    Cobalt.space4,
                  ),
                  children: [
                    ...widget.messages.map(
                      (m) => CoachQuote(text: m.content, role: m.role),
                    ),
                    if (stream.isNotEmpty)
                      CoachQuote(text: stream, role: 'assistant'),
                    for (final finding in findings)
                      if (finding['issue'] is String)
                        CoachQuote(
                          text: finding['issue'] as String,
                          role: 'critic',
                        ),
                  ],
                ),
              ),
            ],
            if (err != null)
              Padding(
                padding: const EdgeInsets.fromLTRB(
                  Cobalt.space6,
                  0,
                  Cobalt.space6,
                  Cobalt.space3,
                ),
                child: CoachNotice(
                  message: localizedError(err, l10n.t),
                  accent: Cobalt.hue200,
                ),
              ),
            if (retry > 0)
              Padding(
                padding: const EdgeInsets.fromLTRB(
                  Cobalt.space6,
                  0,
                  Cobalt.space6,
                  Cobalt.space3,
                ),
                child: CoachNotice(
                  message: l10n.t(
                    err is String && err.contains('unavailable')
                        ? 'coach.serviceUnavailable'
                        : 'coach.rateLimited',
                    {'seconds': retry},
                  ),
                  accent: Cobalt.warnAmber,
                ),
              ),
            if (widget.conversation != null)
              Padding(
                padding: const EdgeInsets.fromLTRB(
                  Cobalt.space4,
                  Cobalt.space2,
                  Cobalt.space4,
                  Cobalt.space4,
                ),
                child: CoachComposer(
                  controller: draft,
                  enabled: !_busy,
                  hint: _busy
                      ? l10n.t('coach.thinking')
                      : l10n.t('coach.placeholder'),
                  sendTooltip: l10n.t('coach.send'),
                  onSend: _busy ? null : () => _send(),
                ),
              ),
          ],
        ),
      ),
    );
  }

  Widget _header(
    BuildContext context,
    Cobalt palette,
    Appearance l10n,
    String stage,
    Color ink,
  ) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(
        Cobalt.space6,
        64,
        Cobalt.space4,
        Cobalt.space2,
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  l10n.t('coach.title').toUpperCase(),
                  style: cobaltSerif(
                    palette,
                    size: 32,
                    color: ink,
                    weight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: Cobalt.space2),
                Text(
                  l10n.t('coach.pullQuotes'),
                  style: cobaltSerif(
                    palette,
                    size: 12,
                    color: ink.withValues(alpha: 0.7),
                    weight: FontWeight.w400,
                  ),
                ),
                if (widget.conversation != null) ...[
                  const SizedBox(height: Cobalt.space2),
                  Text(
                    l10n.t('coach.stage.$stage'),
                    style: cobaltSerif(
                      palette,
                      size: 14,
                      color: ink,
                      weight: FontWeight.w600,
                    ),
                  ),
                ],
                Container(
                  width: Cobalt.space10,
                  height: Cobalt.space1,
                  margin: const EdgeInsets.only(top: Cobalt.space3),
                  color: Cobalt.hue500,
                ),
              ],
            ),
          ),
          const SizedBox(width: Cobalt.space2),
          BlockIconButton(
            icon: Icons.close,
            tooltip: l10n.t('coach.closeCoach'),
            onPressed: () => setState(() => open = false),
          ),
        ],
      ),
    );
  }
}
