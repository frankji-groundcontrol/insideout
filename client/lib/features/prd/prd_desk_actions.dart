import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../theme/block_button.dart';
import '../../theme/cobalt.dart';

/// Mirrors server/internal/store/prds.go prdTransitions: the only legal moves.
const _prdNextStatuses = <String, Set<String>>{
  'draft': {'reviewing'},
  'reviewing': {'approved', 'rejected'},
  'rejected': {'draft'},
};

/// Only a workspace admin approves or rejects (store/prds.go UpdatePrdStatus);
/// draft and reviewing are the author's moves.
const _adminOnly = {'approved', 'rejected'};

class DeskActions extends StatelessWidget {
  const DeskActions({
    super.key,
    required this.prdId,
    required this.status,
    required this.onSave,
    required this.onSnapshot,
    required this.onBuild,
    required this.onStatus,
    this.canDecide = true,
  });

  final String prdId;
  final String? status;

  /// False for a viewer known not to be a workspace admin.
  final bool canDecide;
  final VoidCallback onSave;
  final VoidCallback onSnapshot;
  final VoidCallback onBuild;
  final ValueChanged<String> onStatus;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final palette = Cobalt.of(context);
    final menuStyle = TextStyle(
      fontFamily: Cobalt.sans,
      fontFamilyFallback: Cobalt.sansFallback,
      fontSize: 14,
      fontWeight: FontWeight.w600,
      color: palette.fgPrimary,
    );
    return Wrap(
      spacing: Cobalt.space2,
      runSpacing: Cobalt.space2,
      crossAxisAlignment: WrapCrossAlignment.center,
      children: [
        BlockButton(
          label: l10n.t('common.save'),
          icon: Icons.save_outlined,
          onPressed: onSave,
        ),
        BlockButton(
          label: l10n.t('prd.buildMVP'),
          variant: BlockButtonVariant.secondary,
          icon: Icons.map_outlined,
          onPressed: status == 'approved' ? onBuild : null,
        ),
        BlockButton(
          label: l10n.t('view.entry'),
          variant: BlockButtonVariant.secondary,
          icon: Icons.groups_outlined,
          onPressed: () => context.go('/prd/$prdId/view'),
        ),
        // The menu surface (square, 2px border, no elevation, no ripple) comes
        // from the theme's popupMenuTheme; only the trigger face is local.
        PopupMenuButton<String>(
          tooltip: l10n.t('common.moreActions'),
          child: Container(
            padding: const EdgeInsets.all(Cobalt.space2),
            decoration: BoxDecoration(
              color: palette.surfaceRaised,
              border: Border.all(
                color: palette.strokeSubtle,
                width: Cobalt.borderContainer,
              ),
            ),
            child: Icon(Icons.more_vert, size: 16, color: palette.fgSecondary),
          ),
          onSelected: (value) {
            switch (value) {
              case 'revisions':
                context.go('/prd/$prdId/revisions');
              case 'versions':
                context.go('/prd/$prdId/versions');
              case 'view':
                context.go('/prd/$prdId/view');
              case 'export':
                context.go('/prd/$prdId/export');
              case 'snapshot':
                onSnapshot();
              case 'draft' || 'reviewing' || 'approved' || 'rejected':
                onStatus(value);
            }
          },
          itemBuilder: (_) => [
            _menuItem('revisions', l10n.t('prd.revisionHistory'), menuStyle),
            _menuItem('versions', l10n.t('versions.entry'), menuStyle),
            _menuItem('view', l10n.t('view.entry'), menuStyle),
            _menuItem('export', l10n.t('prd.exportMarkdown'), menuStyle),
            _menuItem('snapshot', l10n.t('prd.snapshotRevision'), menuStyle),
            const PopupMenuDivider(),
            for (final s in const [
              'draft',
              'reviewing',
              'approved',
              'rejected',
            ])
              PopupMenuItem<String>(
                value: s,
                enabled:
                    (_prdNextStatuses[status]?.contains(s) ?? false) &&
                    (canDecide || !_adminOnly.contains(s)),
                child: Text(
                  '${s == status ? '■ ' : ''}${l10n.t('prd.status.$s')}',
                  style: menuStyle,
                ),
              ),
          ],
        ),
      ],
    );
  }
}

PopupMenuItem<String> _menuItem(String value, String label, TextStyle style) =>
    PopupMenuItem<String>(
      value: value,
      child: Text(label, style: style),
    );
