import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/block_field.dart';
import '../../theme/cobalt.dart';
import 'auth_door.dart';
import 'mail_off_notice.dart';

/// `/forgot-password`: requests the emailed password-reset link.
class ForgotPasswordPage extends StatefulWidget {
  const ForgotPasswordPage({super.key});

  @override
  State<ForgotPasswordPage> createState() => _ForgotPasswordPageState();
}

class _ForgotPasswordPageState extends State<ForgotPasswordPage> {
  final email = TextEditingController();
  String? error;
  bool busy = false;
  bool sent = false;

  /// The server's mail word for the sent reset link (PARA-05).
  String? mailWord;
  bool mailOff = false;

  @override
  void initState() {
    super.initState();
    mailIsOff(context).then((off) {
      if (mounted && off) setState(() => mailOff = true);
    });
  }

  @override
  void dispose() {
    email.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    setState(() {
      busy = true;
      error = null;
    });
    final l10n = context.read<Appearance>();
    try {
      final word = await context.read<Session>().api.forgotPassword(
        email.text.trim(),
      );
      if (mounted) {
        setState(() {
          sent = true;
          mailWord = word;
        });
      }
    } catch (e) {
      if (mounted) setState(() => error = l10n.t('forgot.error'));
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Scaffold(
      body: AuthDoor(
        greeting: l10n.t('forgot.title'),
        onClose: () => context.go('/'),
        child: sent
            ? Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    l10n.t('forgot.sent'),
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                  if (mailWord == 'logged') ...[
                    const SizedBox(height: Cobalt.space3),
                    BlockBanner(
                      message: l10n.t('auth.mailLogged'),
                      tone: BlockTone.info,
                    ),
                  ],
                  const SizedBox(height: Cobalt.space6),
                  BlockButton(
                    label: l10n.t('forgot.back'),
                    expand: true,
                    onPressed: () => context.go('/login'),
                  ),
                ],
              )
            : Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  if (mailOff) ...[
                    const MailOffNotice(),
                    const SizedBox(height: Cobalt.space3),
                  ],
                  BlockField(
                    controller: email,
                    label: l10n.t('forgot.email'),
                    autofillHints: const [AutofillHints.email],
                  ),
                  if (error != null) ...[
                    const SizedBox(height: Cobalt.space3),
                    BlockBanner(message: error!),
                  ],
                  const SizedBox(height: Cobalt.space6),
                  BlockButton(
                    label: l10n.t('forgot.submit'),
                    expand: true,
                    busy: busy,
                    onPressed: _submit,
                  ),
                  BlockButton(
                    label: l10n.t('forgot.back'),
                    variant: BlockButtonVariant.ghost,
                    expand: true,
                    onPressed: () => context.go('/login'),
                  ),
                ],
              ),
      ),
    );
  }
}
