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
import 'auth_form_errors.dart';
import 'auth_sent_panels.dart';
import 'mail_off_notice.dart';

/// Login variants offered alongside email+password: an emailed one-time
/// link, or an emailed 6-digit code typed in here.
enum _LoginMode { password, magic, code }

class AuthForm extends StatefulWidget {
  const AuthForm({
    super.key,
    required this.prompt,
    this.inviteCode = '',
    this.onSwitch,
  });

  final AuthPrompt prompt;

  /// Invitation code pre-fill from a `/register?invite=` deep link.
  final String inviteCode;

  final VoidCallback? onSwitch;

  @override
  State<AuthForm> createState() => _AuthFormState();
}

class _AuthFormState extends State<AuthForm> {
  final email = TextEditingController();
  final username = TextEditingController();
  final password = TextEditingController();
  final code = TextEditingController();
  final invite = TextEditingController();
  String? error;
  bool busy = false;
  bool linkSent = false;
  bool codeSent = false;
  bool inviteRequired = false;
  String? signupMode; // from /auth/config; null until it answers
  bool mailOff = false;
  _LoginMode _mode = _LoginMode.password;

  @override
  void initState() {
    super.initState();
    if (widget.prompt == AuthPrompt.register) {
      if (widget.inviteCode.isNotEmpty) {
        invite.text = widget.inviteCode;
      }
      // Public config decides whether the invitation field shows; a fetch
      // failure falls back to hidden so the server's own 403 explains.
      context
          .read<Session>()
          .api
          .authConfig()
          .then((mode) {
            if (mounted) {
              setState(() {
                signupMode = mode;
                inviteRequired = mode == 'invitation_only' || mode == 'closed';
              });
            }
          })
          .catchError((_) {});
    } else {
      mailIsOff(context).then((off) {
        if (mounted && off) setState(() => mailOff = true);
      });
    }
  }

  @override
  void dispose() {
    email.dispose();
    username.dispose();
    password.dispose();
    code.dispose();
    invite.dispose();
    super.dispose();
  }

  bool get _register => widget.prompt == AuthPrompt.register;

  bool get _magic => !_register && _mode == _LoginMode.magic;

  bool get _code => !_register && _mode == _LoginMode.code;

  String get _errorKey {
    if (_magic) return 'login.magicError';
    if (_code) return codeSent ? 'login.codeError' : 'login.codeSendError';
    return _register ? 'register.error' : 'login.error';
  }

  Future<void> _submit() async {
    setState(() {
      busy = true;
      error = null;
    });
    final l10n = context.read<Appearance>();
    try {
      final session = context.read<Session>();
      if (_magic) {
        await session.api.requestMagicLink(email.text.trim());
        if (mounted) setState(() => linkSent = true);
      } else if (_code && !codeSent) {
        await session.api.requestLoginCode(email.text.trim());
        if (mounted) setState(() => codeSent = true);
      } else if (_code) {
        await session.loginWithCode(email.text.trim(), code.text.trim());
        if (mounted) context.go('/dashboard');
      } else if (_register) {
        await session.register(
          email.text.trim(),
          password.text,
          username.text.trim(),
          inviteCode: invite.text.trim(),
        );
        if (mounted) context.go('/dashboard');
      } else {
        await session.login(email.text.trim(), password.text);
        if (mounted) context.go('/dashboard');
      }
    } catch (e) {
      // A register 403 is the door or the invitation, never "try again".
      setState(
        () => error = l10n.t(
          _register ? registerErrorKey(e, signupMode) : _errorKey,
        ),
      );
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  Future<void> _resendCode() async {
    setState(() {
      busy = true;
      error = null;
    });
    final l10n = context.read<Appearance>();
    try {
      await context.read<Session>().api.requestLoginCode(email.text.trim());
    } catch (e) {
      if (mounted) setState(() => error = l10n.t('login.codeSendError'));
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  void _backToPassword() => setState(() {
    _mode = _LoginMode.password;
    linkSent = false;
    codeSent = false;
  });

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    if (_magic && linkSent) {
      return MagicSentPanel(
        message: l10n.t('login.magicSent'),
        backLabel: l10n.t('login.magicBack'),
        onBack: _backToPassword,
      );
    }
    if (_code && codeSent) {
      return CodeEntryPanel(
        message: l10n.t('login.codeSent'),
        controller: code,
        codeLabel: l10n.t('login.codeField'),
        error: error,
        busy: busy,
        verifyLabel: l10n.t('login.codeVerify'),
        resendLabel: l10n.t('login.codeResend'),
        backLabel: l10n.t('login.magicBack'),
        onVerify: _submit,
        onResend: _resendCode,
        onBack: _backToPassword,
      );
    }
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        BlockField(
          controller: email,
          label: l10n.t(_register ? 'register.email' : 'login.username'),
          autofillHints: const [AutofillHints.email],
        ),
        if (_register) ...[
          const SizedBox(height: Cobalt.space3),
          BlockField(controller: username, label: l10n.t('register.username')),
          if (inviteRequired) ...[
            const SizedBox(height: Cobalt.space3),
            BlockField(controller: invite, label: l10n.t('register.invite')),
          ],
        ],
        if (!_magic && !_code) ...[
          const SizedBox(height: Cobalt.space3),
          BlockField(
            controller: password,
            label: l10n.t(_register ? 'register.password' : 'login.password'),
            obscure: true,
            helper: _register ? l10n.t('register.passwordHint') : null,
          ),
        ],
        if (error != null) ...[
          const SizedBox(height: Cobalt.space3),
          BlockBanner(message: error!),
        ],
        const SizedBox(height: Cobalt.space6),
        BlockButton(
          label: l10n.t(switch (_mode) {
            _LoginMode.magic => 'login.magicSubmit',
            _LoginMode.code => 'login.codeSubmit',
            _ => _register ? 'register.submit' : 'login.loginButton',
          }),
          expand: true,
          busy: busy,
          onPressed: _submit,
        ),
        if (_magic || _code) ...[
          const SizedBox(height: Cobalt.space1),
          BlockButton(
            label: l10n.t('login.magicBack'),
            variant: BlockButtonVariant.ghost,
            expand: true,
            onPressed: _backToPassword,
          ),
        ] else if (!_register) ...[
          const SizedBox(height: Cobalt.space6),
          if (mailOff) ...[
            const MailOffNotice(),
            const SizedBox(height: Cobalt.space3),
          ],
          _otherWaysDivider(l10n.t('login.otherWays')),
          const SizedBox(height: Cobalt.space3),
          Row(
            children: [
              Expanded(
                child: BlockButton(
                  label: l10n.t('login.magicLinkShort'),
                  variant: BlockButtonVariant.secondary,
                  expand: true,
                  onPressed: () => setState(() => _mode = _LoginMode.magic),
                ),
              ),
              const SizedBox(width: Cobalt.space3),
              Expanded(
                child: BlockButton(
                  label: l10n.t('login.codeLinkShort'),
                  variant: BlockButtonVariant.secondary,
                  expand: true,
                  onPressed: () => setState(() => _mode = _LoginMode.code),
                ),
              ),
            ],
          ),
          const SizedBox(height: Cobalt.space1),
          Row(
            children: [
              Expanded(
                child: BlockButton(
                  label: l10n.t('login.forgot'),
                  variant: BlockButtonVariant.ghost,
                  size: BlockButtonSize.sm,
                  expand: true,
                  onPressed: () => context.go('/forgot-password'),
                ),
              ),
              Expanded(
                child: BlockButton(
                  label: l10n.t('login.noAccount'),
                  variant: BlockButtonVariant.ghost,
                  size: BlockButtonSize.sm,
                  expand: true,
                  onPressed: widget.onSwitch,
                ),
              ),
            ],
          ),
        ],
        if (_register)
          BlockButton(
            label: l10n.t('register.haveAccount'),
            variant: BlockButtonVariant.ghost,
            expand: true,
            onPressed: widget.onSwitch,
          ),
      ],
    );
  }

  // A hairline "— More ways to sign in —" divider separating the primary
  // password path from the alternate sign-in buttons.
  Widget _otherWaysDivider(String label) {
    final palette = Cobalt.of(context);
    final hairline = Expanded(
      child: Divider(
        thickness: Cobalt.borderDivider,
        height: Cobalt.borderDivider,
        color: palette.strokeSubtle,
      ),
    );
    return Row(
      children: [
        hairline,
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: Cobalt.space3),
          child: Text(
            label,
            style: TextStyle(
              fontFamily: Cobalt.sans,
              fontFamilyFallback: Cobalt.sansFallback,
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color: palette.fgMuted,
            ),
          ),
        ),
        hairline,
      ],
    );
  }
}
