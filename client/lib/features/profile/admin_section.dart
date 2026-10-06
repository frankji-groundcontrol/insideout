import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../../api/errors_l10n.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';

/// Admin surface on the profile page: the signup-mode switch and the
/// invitation ledger (create shows the one-time code once; the server
/// also emails it to the invitee).
class AdminSection extends StatefulWidget {
  const AdminSection({super.key});

  @override
  State<AdminSection> createState() => _AdminSectionState();
}

class _AdminSectionState extends State<AdminSection> {
  final inviteEmail = TextEditingController();
  String? signupMode;
  List<Map<String, dynamic>> invitations = [];
  String? error;
  bool busy = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    inviteEmail.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() => busy = true);
    try {
      final api = context.read<Session>().api;
      final mode = await api.authConfig();
      final list = await api.listInvitations();
      if (mounted) {
        setState(() {
          signupMode = mode;
          invitations = list;
          error = null;
        });
      }
    } catch (e) {
      if (mounted) setState(() => error = e.toString());
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  Future<void> _setMode(String mode) async {
    setState(() => busy = true);
    try {
      await context.read<Session>().api.setSignupMode(mode);
      await _load();
    } catch (e) {
      if (mounted) setState(() => error = e.toString());
      if (mounted) setState(() => busy = false);
    }
  }

  Future<void> _createInvitation() async {
    final email = inviteEmail.text.trim();
    if (email.isEmpty) return;
    setState(() => busy = true);
    try {
      final res = await context.read<Session>().api.createInvitation(email);
      inviteEmail.clear();
      await _load();
      if (!mounted) return;
      final code = res['code']?.toString() ?? '';
      final url = res['signupUrl']?.toString() ?? '';
      showDialog<void>(
        context: context,
        builder: (ctx) => AlertDialog(
          title: Text(l10n.t('admin.codeTitle')),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(l10n.t('admin.codeOnce')),
              const SizedBox(height: 12),
              SelectableText(
                code,
                style: const TextStyle(fontWeight: FontWeight.w700),
              ),
              const SizedBox(height: 8),
              SelectableText(url, style: const TextStyle(fontSize: 12)),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () {
                Clipboard.setData(ClipboardData(text: url));
                Navigator.of(ctx).pop();
              },
              child: Text(l10n.t('admin.copyLink')),
            ),
            TextButton(
              onPressed: () => Navigator.of(ctx).pop(),
              child: const MaterialLocalizationsNo(),
            ),
          ],
        ),
      );
    } catch (e) {
      if (mounted) setState(() => error = e.toString());
      if (mounted) setState(() => busy = false);
    }
  }

  Appearance get l10n => context.read<Appearance>();

  String _modeLabel(String mode) => switch (mode) {
    'open' => l10n.t('admin.modeOpen'),
    'closed' => l10n.t('admin.modeClosed'),
    _ => l10n.t('admin.modeInvitation'),
  };

  @override
  Widget build(BuildContext context) {
    final lang = context.watch<Appearance>();
    if (signupMode == null && busy) {
      return const Padding(
        padding: EdgeInsets.all(16),
        child: CircularProgressIndicator(),
      );
    }
    return Card(
      margin: const EdgeInsets.only(top: 32),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              lang.t('admin.title'),
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              children: [
                for (final mode in const ['invitation_only', 'open', 'closed'])
                  ChoiceChip(
                    label: Text(_modeLabel(mode)),
                    selected: signupMode == mode,
                    onSelected: busy
                        ? null
                        : (sel) {
                            if (sel) _setMode(mode);
                          },
                  ),
              ],
            ),
            const SizedBox(height: 20),
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: inviteEmail,
                    decoration: InputDecoration(
                      labelText: lang.t('admin.inviteEmail'),
                    ),
                    onSubmitted: (_) => _createInvitation(),
                  ),
                ),
                const SizedBox(width: 8),
                FilledButton(
                  onPressed: busy ? null : _createInvitation,
                  child: Text(lang.t('admin.inviteCreate')),
                ),
              ],
            ),
            if (error != null)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Text(
                  localizedError(error!, lang.t),
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                ),
              ),
            const SizedBox(height: 12),
            if (invitations.isNotEmpty) ...[
              Text(
                lang.t('admin.inviteList'),
                style: Theme.of(context).textTheme.bodySmall,
              ),
              const SizedBox(height: 4),
              for (final inv in invitations.take(10))
                ListTile(
                  dense: true,
                  contentPadding: EdgeInsets.zero,
                  title: Text(inv['email']?.toString() ?? ''),
                  subtitle: Text(
                    inv['usedAt'] != null
                        ? lang.t('admin.stateUsed')
                        : inv['revokedAt'] != null
                        ? lang.t('admin.stateRevoked')
                        : lang.t('admin.stateOpen'),
                  ),
                  trailing: inv['usedAt'] == null && inv['revokedAt'] == null
                      ? IconButton(
                          icon: const Icon(Icons.block, size: 18),
                          tooltip: lang.t('admin.revoke'),
                          onPressed: busy
                              ? null
                              : () async {
                                  await context
                                      .read<Session>()
                                      .api
                                      .revokeInvitation(inv['id'].toString());
                                  await _load();
                                },
                        )
                      : null,
                ),
            ],
          ],
        ),
      ),
    );
  }
}

// Tiny close-button label stand-in without needing the full
// MaterialLocalizations lookup in the dialog scope.
class MaterialLocalizationsNo extends StatelessWidget {
  const MaterialLocalizationsNo({super.key});

  @override
  Widget build(BuildContext context) =>
      Text(MaterialLocalizations.of(context).closeButtonLabel);
}
