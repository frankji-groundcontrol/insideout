import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../chrome/time_ago.dart';
import '../../session/appearance.dart';
import '../../theme/block_button.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';
import '../../theme/field_card.dart';

/// One idea on the wall: poster block on a raised-to-sunken gradient (cobalt-950
/// is reserved for status chips, so no slab is tinted with it). The
/// bottom-right disc is the brand mark — a protected identity exception
/// (DESIGN.md), so it stays circular.
class IdeaPoster extends StatelessWidget {
  const IdeaPoster({
    super.key,
    required this.idea,
    required this.onEdit,
    required this.onDrop,
    required this.onConvert,
    required this.onOpenPrd,
  });

  final Idea idea;
  final VoidCallback onEdit;
  final VoidCallback onDrop;
  final VoidCallback onConvert;
  final VoidCallback onOpenPrd;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    // The whole poster does what its button does; the button and the menu
    // keep their own taps, which win over this one.
    final primary = idea.prdId == null ? onConvert : onOpenPrd;
    return FieldCard(
      onTap: primary,
      padding: EdgeInsets.zero,
      child: Container(
        constraints: const BoxConstraints(minHeight: 240),
        decoration: BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [palette.surfaceRaised, palette.surfaceSunken],
          ),
        ),
        child: Stack(
          children: [
            Positioned(
              right: -36,
              bottom: -48,
              child: Container(
                width: 160,
                height: 160,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: palette.brand.withValues(alpha: 0.08),
                ),
              ),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(
                Cobalt.space6,
                Cobalt.space6,
                Cobalt.space3,
                Cobalt.space4,
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          idea.title.toUpperCase(),
                          style: theme.textTheme.headlineSmall,
                        ),
                      ),
                      PopupMenuButton<String>(
                        icon: Icon(Icons.more_vert, color: palette.fgSecondary),
                        tooltip: l10n.t('common.moreActions'),
                        onSelected: (value) {
                          if (value == 'edit') onEdit();
                          if (value == 'drop') onDrop();
                        },
                        itemBuilder: (_) => [
                          PopupMenuItem(
                            value: 'edit',
                            child: Text(l10n.t('common.edit')),
                          ),
                          PopupMenuItem(
                            value: 'drop',
                            child: Text(l10n.t('idea.drop')),
                          ),
                        ],
                      ),
                    ],
                  ),
                  const SizedBox(height: Cobalt.space2),
                  Text(
                    idea.content.isEmpty
                        ? l10n.t('idea.status.${idea.status}')
                        : idea.content,
                    style: theme.textTheme.bodyLarge,
                    maxLines: 4,
                    overflow: TextOverflow.ellipsis,
                  ),
                  const SizedBox(height: Cobalt.space4),
                  ShadeChip(
                    label: l10n.t('idea.touched', {
                      'when': timeAgo(idea.createdAt, l10n.t),
                    }),
                    tone: ShadeTone.neutral,
                  ),
                  const SizedBox(height: Cobalt.space3),
                  Align(
                    alignment: Alignment.centerLeft,
                    child: BlockButton(
                      label: idea.prdId == null
                          ? l10n.t('idea.convert')
                          : l10n.t('prd.title'),
                      size: BlockButtonSize.sm,
                      variant: idea.prdId == null
                          ? BlockButtonVariant.primary
                          : BlockButtonVariant.secondary,
                      trailingIcon: Icons.arrow_forward,
                      onPressed: primary,
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
