import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_chrome.dart';

/// Whether this server only logs email (mail dev mode). A failed call answers
/// false: the emailed options are annotated when the server says so, never on
/// a guess.
Future<bool> mailIsOff(BuildContext context) async {
  final api = context.read<Session>().api;
  try {
    return !await api.authMailEnabled();
  } catch (_) {
    return false;
  }
}

/// Tells the reader that sign-in links, codes and password-reset mails are
/// only written to the server log. It annotates the options; it never hides
/// them.
class MailOffNotice extends StatelessWidget {
  const MailOffNotice({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return BlockBanner(message: l10n.t('auth.mailOff'), tone: BlockTone.info);
  }
}
