import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../session/session.dart';
import 'auth_door.dart';

/// Login variants offered alongside email+password: an emailed one-time
/// link, or an emailed 6-digit code typed in here.
enum _LoginMode { password, magic, code }

class AuthForm extends StatefulWidget {
  const AuthForm({super.key, required this.prompt, this.onSwitch});

  final AuthPrompt prompt;
  final VoidCallback? onSwitch;

  @override
  State<AuthForm> createState() => _AuthFormState();
}

class _AuthFormState extends State<AuthForm> {
  final email = TextEditingController();
  final username = TextEditingController();
  final password = TextEditingController();
  final code = TextEditingController();
  String? error;
  bool busy = false;
  bool linkSent = false;
  bool codeSent = false;
  _LoginMode _mode = _LoginMode.password;

  @override
  void dispose() {
    email.dispose();
    username.dispose();
    password.dispose();
    code.dispose();
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
        await session.register(email.text.trim(), password.text, username.text.trim());
        if (mounted) context.go('/dashboard');
      } else {
        await session.login(email.text.trim(), password.text);
        if (mounted) context.go('/dashboard');
      }
    } catch (e) {
      setState(() => error = l10n.t(_errorKey));
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
      return Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(l10n.t('login.magicSent'), style: Theme.of(context).textTheme.bodyMedium),
          const SizedBox(height: 20),
          TextButton(
            onPressed: _backToPassword,
            child: Text(l10n.t('login.magicBack')),
          ),
        ],
      );
    }
    if (_code && codeSent) {
      return Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(l10n.t('login.codeSent'), style: Theme.of(context).textTheme.bodyMedium),
          const SizedBox(height: 16),
          TextField(
            controller: code,
            decoration: InputDecoration(labelText: l10n.t('login.codeField')),
            keyboardType: TextInputType.number,
            maxLength: 6,
            autofocus: true,
          ),
          if (error != null) ...[
            const SizedBox(height: 12),
            Text(error!, style: TextStyle(color: Theme.of(context).colorScheme.error)),
          ],
          const SizedBox(height: 20),
          FilledButton(
            onPressed: busy ? null : _submit,
            child: Text(l10n.t('login.codeVerify')),
          ),
          const SizedBox(height: 4),
          TextButton(
            onPressed: busy ? null : _resendCode,
            child: Text(l10n.t('login.codeResend')),
          ),
          TextButton(
            onPressed: _backToPassword,
            child: Text(l10n.t('login.magicBack')),
          ),
        ],
      );
    }
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        TextField(
          controller: email,
          decoration: InputDecoration(labelText: l10n.t(_register ? 'register.email' : 'login.username')),
          autofillHints: const [AutofillHints.email],
        ),
        if (_register) ...[
          const SizedBox(height: 12),
          TextField(controller: username, decoration: InputDecoration(labelText: l10n.t('register.username'))),
        ],
        if (!_magic && !_code) ...[
          const SizedBox(height: 12),
          TextField(
            controller: password,
            decoration: InputDecoration(
              labelText: l10n.t(_register ? 'register.password' : 'login.password'),
              helperText: _register ? l10n.t('register.passwordHint') : null,
            ),
            obscureText: true,
          ),
        ],
        if (error != null) ...[
          const SizedBox(height: 12),
          Text(error!, style: TextStyle(color: Theme.of(context).colorScheme.error)),
        ],
        const SizedBox(height: 20),
        FilledButton(
          onPressed: busy ? null : _submit,
          child: Text(l10n.t(switch (_mode) {
            _LoginMode.magic => 'login.magicSubmit',
            _LoginMode.code => 'login.codeSubmit',
            _ => _register ? 'register.submit' : 'login.loginButton',
          })),
        ),
        if (_magic || _code) ...[
          const SizedBox(height: 4),
          TextButton(
            onPressed: _backToPassword,
            child: Text(l10n.t('login.magicBack')),
          ),
        ] else if (!_register) ...[
          const SizedBox(height: 4),
          TextButton(
            onPressed: () => setState(() => _mode = _LoginMode.magic),
            child: Text(l10n.t('login.magicLink')),
          ),
          TextButton(
            onPressed: () => setState(() => _mode = _LoginMode.code),
            child: Text(l10n.t('login.codeLink')),
          ),
          TextButton(
            onPressed: () => context.go('/forgot-password'),
            child: Text(l10n.t('login.forgot')),
          ),
        ],
        TextButton(
          onPressed: widget.onSwitch,
          child: Text(l10n.t(_register ? 'register.haveAccount' : 'login.noAccount')),
        ),
      ],
    );
  }
}
