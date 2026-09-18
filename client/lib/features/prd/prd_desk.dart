import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import '../../theme/field_card.dart';

/// Lock D — left half of the split desk: large type, section cards, not type-rows.
class PrdDeskColumn extends StatelessWidget {
  const PrdDeskColumn({
    super.key,
    required this.prdId,
    required this.prd,
    required this.titleCtrl,
    required this.sectionCtrls,
    required this.onSave,
    required this.onSaveTitle,
    required this.onRequestReview,
    required this.onSnapshot,
    required this.onBuild,
    required this.onStatus,
  });

  final String prdId;
  final Prd? prd;
  final TextEditingController titleCtrl;
  final Map<String, TextEditingController> sectionCtrls;
  final VoidCallback onSave;
  final VoidCallback onSaveTitle;
  final VoidCallback onRequestReview;
  final VoidCallback onSnapshot;
  final VoidCallback onBuild;
  final ValueChanged<String> onStatus;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    return ListView(
      padding: const EdgeInsets.fromLTRB(28, 64, 24, 140),
      children: [
        Text(l10n.t('prd.pageTitle').toUpperCase(), style: theme.textTheme.displaySmall),
        const SizedBox(height: 20),
        Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(
              child: TextField(
                controller: titleCtrl,
                style: theme.textTheme.headlineLarge,
                decoration: const InputDecoration(
                  border: InputBorder.none,
                  enabledBorder: InputBorder.none,
                  focusedBorder: InputBorder.none,
                  filled: false,
                  isDense: true,
                ),
                onSubmitted: (_) => onSaveTitle(),
              ),
            ),
            FilledButton(onPressed: onRequestReview, child: Text(l10n.t('prd.requestReview'))),
          ],
        ),
        const SizedBox(height: 4),
        Text(l10n.t('prd.workingVersion'), style: theme.textTheme.titleMedium),
        Container(width: 48, height: 3, margin: const EdgeInsets.only(top: 8, bottom: 12), color: palette.brand),
        Text(
          'v${prd?.currentRevision ?? 0} · ${prd?.updatedAt ?? ''}',
          style: theme.textTheme.bodySmall,
        ),
        const SizedBox(height: 12),
        _DeskActions(
          prdId: prdId,
          status: prd?.status,
          onSave: onSave,
          onSnapshot: onSnapshot,
          onBuild: onBuild,
          onStatus: onStatus,
        ),
        const SizedBox(height: 20),
        for (final key in prdSectionKeys)
          Padding(
            padding: const EdgeInsets.only(bottom: 16),
            child: FieldCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(l10n.t('prd.sections.$key'), style: theme.textTheme.titleLarge),
                  const SizedBox(height: 8),
                  TextField(
                    controller: sectionCtrls[key],
                    maxLines: null,
                    minLines: 3,
                    style: theme.textTheme.bodyLarge,
                    decoration: const InputDecoration(
                      border: InputBorder.none,
                      enabledBorder: InputBorder.none,
                      focusedBorder: InputBorder.none,
                      filled: false,
                    ),
                  ),
                ],
              ),
            ),
          ),
      ],
    );
  }
}

class _DeskActions extends StatelessWidget {
  const _DeskActions({
    required this.prdId,
    required this.status,
    required this.onSave,
    required this.onSnapshot,
    required this.onBuild,
    required this.onStatus,
  });

  final String prdId;
  final String? status;
  final VoidCallback onSave;
  final VoidCallback onSnapshot;
  final VoidCallback onBuild;
  final ValueChanged<String> onStatus;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      crossAxisAlignment: WrapCrossAlignment.center,
      children: [
        FilledButton(onPressed: onSave, child: Text(l10n.t('common.save'))),
        OutlinedButton(onPressed: onBuild, child: Text(l10n.t('prd.buildMVP'))),
        PopupMenuButton<String>(
          tooltip: l10n.t('prd.moreActions'),
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
            PopupMenuItem(value: 'revisions', child: Text(l10n.t('prd.revisionHistory'))),
            PopupMenuItem(value: 'versions', child: Text(l10n.t('versions.entry'))),
            PopupMenuItem(value: 'view', child: Text(l10n.t('view.entry'))),
            PopupMenuItem(value: 'export', child: Text(l10n.t('prd.exportMarkdown'))),
            PopupMenuItem(value: 'snapshot', child: Text(l10n.t('prd.snapshotRevision'))),
            const PopupMenuDivider(),
            for (final s in const ['draft', 'reviewing', 'approved', 'rejected'])
              PopupMenuItem(value: s, child: Text('${s == status ? '● ' : ''}${l10n.t('prd.status.$s')}')),
          ],
        ),
      ],
    );
  }
}
