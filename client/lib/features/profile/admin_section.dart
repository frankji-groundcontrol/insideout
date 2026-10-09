import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../../api/errors_l10n.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/block_field.dart';
import '../../theme/block_modal.dart';
import '../../theme/block_state.dart';
import '../../theme/cobalt.dart';
import '../../theme/field_card.dart';

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
  Object? error;
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
      if (mounted) setState(() => error = e);
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
      if (mounted) setState(() => error = e);
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
      await showBlockModal<void>(
        context: context,
        title: l10n.t('admin.codeTitle'),
        body: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(l10n.t('admin.codeOnce')),
            const SizedBox(height: Cobalt.space3),
            SelectableText(
              code,
              style: const TextStyle(fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: Cobalt.space2),
            SelectableText(url, style: const TextStyle(fontSize: 12)),
          ],
        ),
        footer: [
          BlockButton(
            label: l10n.t('admin.copyLink'),
            icon: Icons.content_copy,
            variant: BlockButtonVariant.secondary,
            onPressed: () {
              Clipboard.setData(ClipboardData(text: url));
              Navigator.of(context).pop();
            },
          ),
        ],
      );
    } catch (e) {
      if (mounted) setState(() => error = e);
      if (mounted) setState(() => busy = false);
    }
  }

  Future<void> _revoke(Map<String, dynamic> inv) async {
    final api = context.read<Session>().api;
    final ok = await showBlockConfirm(
      context: context,
      title: l10n.t('admin.revoke'),
      message: l10n.t('admin.revokeConfirm'),
      confirmText: l10n.t('admin.revoke'),
      danger: true,
    );
    if (!ok) return;
    setState(() => busy = true);
    try {
      await api.revokeInvitation(inv['id'].toString());
      await _load();
    } catch (e) {
      if (mounted) setState(() => error = e);
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
        padding: EdgeInsets.all(Cobalt.space4),
        child: BlockLoading(),
      );
    }
    if (signupMode == null && error != null) {
      return BlockError(error: error, onRetry: _load);
    }
    return Padding(
      padding: const EdgeInsets.only(top: Cobalt.space8),
      child: FieldCard(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              lang.t('admin.title'),
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: Cobalt.space4),
            Wrap(
              spacing: Cobalt.space2,
              runSpacing: Cobalt.space2,
              children: [
                for (final mode in const ['invitation_only', 'open', 'closed'])
                  BlockButton(
                    label: _modeLabel(mode),
                    size: BlockButtonSize.sm,
                    variant: signupMode == mode
                        ? BlockButtonVariant.primary
                        : BlockButtonVariant.secondary,
                    onPressed: busy ? null : () => _setMode(mode),
                  ),
              ],
            ),
            const SizedBox(height: Cobalt.space6),
            Row(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Expanded(
                  child: BlockField(
                    controller: inviteEmail,
                    label: lang.t('admin.inviteEmail'),
                    onSubmitted: (_) => _createInvitation(),
                  ),
                ),
                const SizedBox(width: Cobalt.space2),
                BlockButton(
                  label: lang.t('admin.inviteCreate'),
                  busy: busy,
                  onPressed: _createInvitation,
                ),
              ],
            ),
            if (error != null) ...[
              const SizedBox(height: Cobalt.space3),
              BlockBanner(message: localizedError(error!, lang.t)),
            ],
            if (invitations.isNotEmpty) ...[
              const SizedBox(height: Cobalt.space4),
              Text(
                lang.t('admin.inviteList'),
                style: Theme.of(context).textTheme.bodySmall,
              ),
              const SizedBox(height: Cobalt.space2),
              for (final inv in invitations.take(10))
                _InvitationRow(
                  invitation: inv,
                  busy: busy,
                  onRevoke: () => _revoke(inv),
                ),
            ],
          ],
        ),
      ),
    );
  }
}

class _InvitationRow extends StatelessWidget {
  const _InvitationRow({
    required this.invitation,
    required this.busy,
    required this.onRevoke,
  });

  final Map<String, dynamic> invitation;
  final bool busy;
  final VoidCallback onRevoke;

  @override
  Widget build(BuildContext context) {
    final lang = context.watch<Appearance>();
    final open =
        invitation['usedAt'] == null && invitation['revokedAt'] == null;
    final (stateLabel, tone) = invitation['usedAt'] != null
        ? (lang.t('admin.stateUsed'), BlockTone.neutral)
        : invitation['revokedAt'] != null
        ? (lang.t('admin.stateRevoked'), BlockTone.neutral)
        : (lang.t('admin.stateOpen'), BlockTone.info);
    return Padding(
      padding: const EdgeInsets.only(bottom: Cobalt.space2),
      child: Row(
        children: [
          Expanded(
            child: Text(
              invitation['email']?.toString() ?? '',
              overflow: TextOverflow.ellipsis,
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ),
          const SizedBox(width: Cobalt.space2),
          BlockBadge(label: stateLabel, tone: tone),
          if (open) ...[
            const SizedBox(width: Cobalt.space2),
            BlockIconButton(
              icon: Icons.block,
              tooltip: lang.t('admin.revoke'),
              semanticLabel: lang.t('admin.revoke'),
              onPressed: busy ? null : onRevoke,
            ),
          ],
        ],
      ),
    );
  }
}
