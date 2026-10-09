import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../secret_query_scrub.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/block_field.dart';
import '../../theme/cobalt.dart';
import 'auth_door.dart';

/// `/reset-password?token=...`: the landing page for the emailed reset
/// link. Sets a new password; every existing session was revoked
/// server-side, so the only next step is a fresh login.
class ResetPasswordPage extends StatefulWidget {
  const ResetPasswordPage({super.key, required this.token});

  final String token;

  @override
  State<ResetPasswordPage> createState() => _ResetPasswordPageState();
}

class _ResetPasswordPageState extends State<ResetPasswordPage> {
  final password = TextEditingController();
  final confirm = TextEditingController();
  String? error;
  bool busy = false;
  bool done = false;
  late final String _token;

  @override
  void initState() {
    super.initState();
    _token = widget.token;
    scrubSecretQueryKeys(const ['token']);
  }

  @override
  void dispose() {
    password.dispose();
    confirm.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final l10n = context.read<Appearance>();
    if (password.text != confirm.text) {
      setState(() => error = l10n.t('reset.mismatch'));
      return;
    }
    setState(() {
      busy = true;
      error = null;
    });
    try {
      await context.read<Session>().api.resetPassword(_token, password.text);
      if (mounted) setState(() => done = true);
    } catch (e) {
      if (mounted) setState(() => error = l10n.t('reset.error'));
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Scaffold(
      body: AuthDoor(
        greeting: l10n.t('reset.title'),
        onClose: () => context.go('/'),
        child: done
            ? Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    l10n.t('reset.done'),
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                  const SizedBox(height: Cobalt.space6),
                  BlockButton(
                    label: l10n.t('reset.doneLogin'),
                    expand: true,
                    onPressed: () => context.go('/login'),
                  ),
                ],
              )
            : Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  BlockField(
                    controller: password,
                    label: l10n.t('reset.password'),
                    obscure: true,
                    helper: l10n.t('reset.passwordHint'),
                  ),
                  const SizedBox(height: Cobalt.space3),
                  BlockField(
                    controller: confirm,
                    label: l10n.t('reset.confirm'),
                    obscure: true,
                  ),
                  if (error != null) ...[
                    const SizedBox(height: Cobalt.space3),
                    BlockBanner(message: error!),
                  ],
                  const SizedBox(height: Cobalt.space6),
                  BlockButton(
                    label: l10n.t('reset.submit'),
                    expand: true,
                    busy: busy,
                    onPressed: _submit,
                  ),
                  BlockButton(
                    label: l10n.t('reset.back'),
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
