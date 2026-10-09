import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../theme/block_button.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';
import '../../theme/field_card.dart';
import 'prd_desk_actions.dart';

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
    final status = prd?.status ?? 'draft';
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: Cobalt.contentMaxWidth),
        child: ListView(
          padding: const EdgeInsets.fromLTRB(
            Cobalt.space6,
            64,
            Cobalt.space6,
            140,
          ),
          children: [
            Text(
              l10n.t('prd.pageTitle').toUpperCase(),
              style: theme.textTheme.displaySmall,
            ),
            const SizedBox(height: Cobalt.space6),
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: _FlatEditor(
                    controller: titleCtrl,
                    style: theme.textTheme.headlineLarge!,
                    maxLines: 1,
                    onSubmitted: (_) => onSaveTitle(),
                  ),
                ),
                const SizedBox(width: Cobalt.space3),
                BlockButton(
                  label: status == 'draft'
                      ? l10n.t('prd.next.requestReview')
                      : l10n.t('prd.requestReview'),
                  icon: Icons.arrow_forward,
                  onPressed: status == 'draft' ? onRequestReview : null,
                ),
              ],
            ),
            const SizedBox(height: Cobalt.space6),
            Text(
              l10n.t('prd.workingVersion'),
              style: theme.textTheme.titleMedium,
            ),
            Container(
              width: Cobalt.space10,
              height: Cobalt.space1,
              margin: const EdgeInsets.only(
                top: Cobalt.space2,
                bottom: Cobalt.space3,
              ),
              color: palette.brand,
            ),
            Row(
              children: [
                ShadeChip(
                  label: l10n.t('prd.status.$status'),
                  tone: _statusTone(status),
                ),
                const SizedBox(width: Cobalt.space3),
                Expanded(
                  child: Text(
                    'v${prd?.currentRevision ?? 0} · ${prd?.updatedAt ?? ''}',
                    style: theme.textTheme.bodySmall,
                  ),
                ),
              ],
            ),
            const SizedBox(height: Cobalt.space4),
            DeskActions(
              prdId: prdId,
              status: prd?.status,
              onSave: onSave,
              onSnapshot: onSnapshot,
              onBuild: onBuild,
              onStatus: onStatus,
            ),
            const SizedBox(height: Cobalt.space6),
            for (final key in prdSectionKeys)
              Padding(
                padding: const EdgeInsets.only(bottom: Cobalt.space4),
                child: FieldCard(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        l10n.t('prd.sections.$key'),
                        style: theme.textTheme.titleLarge,
                      ),
                      const SizedBox(height: Cobalt.space2),
                      Divider(
                        height: Cobalt.borderDivider,
                        thickness: Cobalt.borderDivider,
                        color: palette.strokeSubtle,
                      ),
                      const SizedBox(height: Cobalt.space2),
                      _FlatEditor(
                        controller: sectionCtrls[key],
                        style: theme.textTheme.bodyLarge!,
                        minLines: 3,
                        maxLines: null,
                      ),
                    ],
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

ShadeTone _statusTone(String status) => switch (status) {
  'approved' => ShadeTone.done,
  'reviewing' => ShadeTone.progress,
  'rejected' => ShadeTone.danger,
  _ => ShadeTone.pending,
};

/// In-card editor: the card is the block, so the field stays flat and lights a
/// cobalt edge on focus instead of stacking a second border and cast inside it.
class _FlatEditor extends StatefulWidget {
  const _FlatEditor({
    required this.controller,
    required this.style,
    this.minLines = 1,
    this.maxLines = 1,
    this.onSubmitted,
  });

  final TextEditingController? controller;
  final TextStyle style;
  final int minLines;
  final int? maxLines;
  final ValueChanged<String>? onSubmitted;

  @override
  State<_FlatEditor> createState() => _FlatEditorState();
}

class _FlatEditorState extends State<_FlatEditor> {
  final node = FocusNode();

  @override
  void initState() {
    super.initState();
    node.addListener(_repaint);
  }

  void _repaint() {
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    node
      ..removeListener(_repaint)
      ..dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return Container(
      decoration: BoxDecoration(
        border: Border.all(
          color: node.hasFocus ? palette.brand : Colors.transparent,
          width: Cobalt.borderContainer,
        ),
      ),
      padding: const EdgeInsets.symmetric(
        horizontal: Cobalt.space2,
        vertical: Cobalt.space1,
      ),
      child: TextField(
        controller: widget.controller,
        focusNode: node,
        minLines: widget.minLines,
        maxLines: widget.maxLines,
        style: widget.style.copyWith(color: palette.fgPrimary),
        onSubmitted: widget.onSubmitted,
        decoration: const InputDecoration(
          border: InputBorder.none,
          enabledBorder: InputBorder.none,
          focusedBorder: InputBorder.none,
          filled: false,
          isDense: true,
          contentPadding: EdgeInsets.zero,
        ),
      ),
    );
  }
}
