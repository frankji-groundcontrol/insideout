import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import 'coach_trust_notes.dart';

/// Transient notices for what the coach records or skips as a turn streams.
/// The caller passes a live context: a frame that lands after the panel is
/// gone is dropped before it gets here.
void noticeFactRecorded(BuildContext context, Map<String, dynamic> fact) {
  // The payload is {id, kind, text, status}: show the text alone.
  final text = fact['text'];
  if (text is! String || text.isEmpty) return;
  _snack(
    context,
    context.read<Appearance>().t('coach.factRecorded', {'fact': text}),
  );
}

/// A critic pass did not run (rv4-AIDEV-02): say so as it happens, in the same
/// words the finalize card uses.
void noticeCriticSkipped(BuildContext context, String reason, int rounds) {
  final note = criticSkipNote(context.read<Appearance>(), reason, rounds);
  if (note != null) _snack(context, note);
}

void _snack(BuildContext context, String text) =>
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(text)));
