import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/errors.dart';
import '../../api/errors_l10n.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_field.dart';
import '../../theme/block_modal.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_theme.dart';

/// The profile page's danger block: deletes the signed-in account after the
/// current password, clears the session and returns to the landing page
/// (FE-07; the route is in docs/usage/account-deletion.md).
class DeleteAccountSection extends StatelessWidget {
  const DeleteAccountSection({super.key});

  Future<void> _delete(BuildContext context) async {
    final t = context.read<Appearance>().t;
    final session = context.read<Session>();
    final messenger = ScaffoldMessenger.of(context);
    final navigator = Navigator.of(context);
    // Not disposed: the pop animation keeps the field mounted past the await.
    final password = TextEditingController();
    final ok = await showBlockModal<bool>(
      context: context,
      title: t('profile.deleteAccount'),
      body: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(t('profile.deleteConfirm')),
          const SizedBox(height: Cobalt.space4),
          BlockField(
            controller: password,
            label: t('profile.currentPassword'),
            obscure: true,
          ),
        ],
      ),
      footer: [
        BlockButton(
          label: t('common.cancel'),
          variant: BlockButtonVariant.secondary,
          onPressed: () => navigator.pop(false),
        ),
        BlockButton(
          label: t('profile.deleteForever'),
          variant: BlockButtonVariant.danger,
          onPressed: () => navigator.pop(true),
        ),
      ],
    );
    if (ok != true || password.text.isEmpty) return;
    try {
      await session.api.deleteMe(password.text);
    } catch (e) {
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            e is ApiException && e.status == 403
                ? t('profile.deleteWrongPassword')
                : localizedError(e, t),
          ),
        ),
      );
      return;
    }
    await session.clear();
    if (context.mounted) context.go('/');
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Padding(
      padding: const EdgeInsets.only(top: Cobalt.space8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.t('profile.dangerTitle').toUpperCase(),
            style: eyebrow(context),
          ),
          const SizedBox(height: Cobalt.space2),
          Text(
            l10n.t('profile.deleteHint'),
            style: Theme.of(context).textTheme.bodySmall,
          ),
          const SizedBox(height: Cobalt.space3),
          BlockButton(
            label: l10n.t('profile.deleteAccount'),
            icon: Icons.delete_forever_outlined,
            variant: BlockButtonVariant.danger,
            onPressed: () => _delete(context),
          ),
        ],
      ),
    );
  }
}
