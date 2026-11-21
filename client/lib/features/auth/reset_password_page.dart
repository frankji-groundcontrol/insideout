import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/errors.dart';
import '../../api/errors_l10n.dart';
import '../../secret_query_scrub.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/block_field.dart';
import '../../theme/cobalt.dart';
import 'auth_door.dart';

/// `/reset-password?token=...`: the landing page for the emailed reset
/// link. The person can set a new password here, or copy the code and
/// finish with `insideout reset-password`. Either path uses the code once.
class ResetPasswordPage extends StatefulWidget {
  const ResetPasswordPage({super.key, required this.token});

  final String token;

  @override
  State<ResetPasswordPage> createState() => _ResetPasswordPageState();
}

enum _ResetStep { choose, web, cli, done }

class _ResetPasswordPageState extends State<ResetPasswordPage> {
  final password = TextEditingController();
  final confirm = TextEditingController();
  String? error;
  bool busy = false;
  bool copied = false;
  _ResetStep step = _ResetStep.choose;
  late final String _token;

  @override
  void initState() {
    super.initState();
    _token = widget.token;
    scrubSecretQueryKeys(const ['token']);
    if (_token.isEmpty) step = _ResetStep.web;
  }

  @override
  void dispose() {
    password.dispose();
    confirm.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final l10n = context.read<Appearance>();
    if (password.text.length < 8) {
      setState(() => error = l10n.t('reset.tooShort'));
      return;
    }
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
      if (mounted) setState(() => step = _ResetStep.done);
    } catch (e) {
      if (mounted) setState(() => error = _failure(e, l10n));
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  /// Only a refused token is a dead link: a throttle or a lost connection
  /// leaves the link usable, so saying it expired would send people away.
  String _failure(Object e, Appearance l10n) {
    if (e is ApiException && e.status == 429) return l10n.t('reset.throttled');
    if (e is ApiException && (e.status ?? 500) < 500) {
      return l10n.t('reset.error');
    }
    final serverFault =
        e is DioException && (e.response?.statusCode ?? 0) >= 500;
    if (!serverFault && (e is DioException || e is TimeoutException)) {
      return l10n.t('reset.network');
    }
    return localizedError(e, l10n.t);
  }

  Future<void> _copy() async {
    await Clipboard.setData(ClipboardData(text: _token));
    if (mounted) setState(() => copied = true);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Scaffold(
      body: AuthDoor(
        greeting: l10n.t('reset.title'),
        onClose: () => context.go('/'),
        child: switch (step) {
          _ResetStep.done => _done(context, l10n),
          _ResetStep.choose => _choose(l10n),
          _ResetStep.cli => _cli(context, l10n),
          _ResetStep.web => _web(l10n),
        },
      ),
    );
  }

  Widget _choose(Appearance l10n) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        BlockButton(
          label: l10n.t('reset.chooseWeb'),
          expand: true,
          onPressed: () => setState(() => step = _ResetStep.web),
        ),
        BlockButton(
          label: l10n.t('reset.chooseCli'),
          variant: BlockButtonVariant.secondary,
          expand: true,
          onPressed: () => setState(() => step = _ResetStep.cli),
        ),
        BlockButton(
          label: l10n.t('reset.back'),
          variant: BlockButtonVariant.ghost,
          expand: true,
          onPressed: () => context.go('/login'),
        ),
      ],
    );
  }

  Widget _cli(BuildContext context, Appearance l10n) {
    final style = Theme.of(context).textTheme;
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(l10n.t('reset.cliHint'), style: style.bodyMedium),
        const SizedBox(height: Cobalt.space4),
        SelectableText(_token, style: style.titleMedium),
        const SizedBox(height: Cobalt.space3),
        Text(l10n.t('reset.cliCommand'), style: style.bodySmall),
        if (copied) ...[
          const SizedBox(height: Cobalt.space3),
          Text(l10n.t('reset.cliCopied'), style: style.bodyMedium),
        ],
        const SizedBox(height: Cobalt.space6),
        BlockButton(
          label: l10n.t('reset.cliCopy'),
          expand: true,
          onPressed: _copy,
        ),
        BlockButton(
          label: l10n.t('reset.cliBack'),
          variant: BlockButtonVariant.ghost,
          expand: true,
          onPressed: () => setState(() {
            copied = false;
            step = _ResetStep.choose;
          }),
        ),
      ],
    );
  }

  Widget _web(Appearance l10n) {
    return Column(
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
          onPressed: _token.isEmpty ? null : _submit,
        ),
        BlockButton(
          label: _token.isEmpty
              ? l10n.t('reset.back')
              : l10n.t('reset.cliBack'),
          variant: BlockButtonVariant.ghost,
          expand: true,
          onPressed: () {
            if (_token.isEmpty) {
              context.go('/login');
            } else {
              setState(() {
                error = null;
                step = _ResetStep.choose;
              });
            }
          },
        ),
      ],
    );
  }

  Widget _done(BuildContext context, Appearance l10n) {
    return Column(
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
    );
  }
}
