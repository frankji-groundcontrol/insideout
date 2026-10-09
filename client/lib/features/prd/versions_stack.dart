import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/cobalt.dart';
import '../../theme/field_card.dart';
import '../../widgets/pressable_block.dart';
import 'audience_selector.dart';
import 'versions_commit_dialog.dart' show versionAudiences;

/// Lock C — centered fanned commit stack, not a 300px list + detail row.
class VersionsStack extends StatelessWidget {
  const VersionsStack({
    super.key,
    required this.prdId,
    required this.commits,
    required this.selected,
    required this.readiness,
    required this.audience,
    required this.onSelect,
    required this.onAudience,
    required this.onCommit,
  });

  final String prdId;
  final List<PrdCommit> commits;
  final PrdCommit? selected;
  final PrdReadiness? readiness;
  final String audience;
  final void Function(String id) onSelect;
  final void Function(String audience) onAudience;
  final VoidCallback onCommit;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final behind = commits.where((c) => c.id != selected?.id).take(2).toList();
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: Cobalt.contentMaxWidth),
        child: ListView(
          padding: const EdgeInsets.fromLTRB(
            Cobalt.space6,
            56,
            Cobalt.space6,
            140,
          ),
          children: [
            if (readiness != null)
              AudienceSelector(
                audiences: versionAudiences,
                selected: audience,
                onSelected: onAudience,
                labels: (a) => l10n.t('versions.audience.$a'),
              ),
            const SizedBox(height: Cobalt.space4),
            Text(
              l10n.t('versions.stackEyebrow', {
                'n': selected?.revision ?? 0,
                'total': commits.length,
              }),
              textAlign: TextAlign.center,
              style: theme.textTheme.labelSmall,
            ),
            const SizedBox(height: Cobalt.space6),
            Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 640),
                child: SizedBox(
                  height: 580,
                  child: Stack(
                    alignment: Alignment.topCenter,
                    children: [
                      for (var i = 0; i < behind.length; i++)
                        Positioned(
                          top: i * 36.0,
                          left: Cobalt.space8 + i * Cobalt.space2,
                          right: Cobalt.space8 + i * Cobalt.space2,
                          child: _PeekCard(
                            name: behind[i].name,
                            onTap: () => onSelect(behind[i].id),
                          ),
                        ),
                      Positioned(
                        top: behind.length * Cobalt.controlHeight,
                        left: 0,
                        right: 0,
                        bottom: 0,
                        child: _FrontCard(
                          prdId: prdId,
                          commit: selected,
                          readiness: readiness,
                          audience: audience,
                          onCommit: onCommit,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// A commit peeking out from behind the front card — its cobalt tab is the
/// handle that brings it forward.
class _PeekCard extends StatelessWidget {
  const _PeekCard({required this.name, required this.onTap});

  final String name;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return PressableBlock(
      onTap: onTap,
      child: Column(
        children: [
          Container(
            width: double.infinity,
            padding: const EdgeInsets.symmetric(
              horizontal: Cobalt.space3,
              vertical: Cobalt.space2,
            ),
            color: palette.brand,
            child: Text(
              name,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(
                context,
              ).textTheme.labelSmall?.copyWith(color: palette.btnFg),
            ),
          ),
          FieldCard(
            color: palette.brandSubtle,
            padding: const EdgeInsets.symmetric(vertical: Cobalt.space4),
            child: const SizedBox(
              height: Cobalt.space2,
              width: double.infinity,
            ),
          ),
        ],
      ),
    );
  }
}

class _FrontCard extends StatelessWidget {
  const _FrontCard({
    required this.prdId,
    required this.commit,
    required this.readiness,
    required this.audience,
    required this.onCommit,
  });

  final String prdId;
  final PrdCommit? commit;
  final PrdReadiness? readiness;
  final String audience;
  final VoidCallback onCommit;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    final ar = readiness?.audiences[audience];
    return FieldCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const BlockIconWell(icon: Icons.verified, size: 40, iconSize: 20),
              const SizedBox(width: Cobalt.space3),
              if (commit == null)
                Expanded(
                  child: Text(
                    l10n.t('versions.empty'),
                    style: theme.textTheme.headlineSmall,
                  ),
                )
              else ...[
                Text(
                  '${commit!.revision}',
                  style: theme.textTheme.headlineLarge,
                ),
                const SizedBox(width: Cobalt.space3),
                Expanded(
                  child: Text(
                    commit!.name,
                    style: theme.textTheme.headlineSmall,
                  ),
                ),
              ],
            ],
          ),
          const SizedBox(height: Cobalt.space4),
          // The readiness list and the commit details flex and scroll inside
          // the card so a long gap list can never push the Commit action off
          // the fanned stack.
          Flexible(
            child: SingleChildScrollView(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  if (ar != null) ...[
                    Text(
                      ar.ready
                          ? l10n.t('versions.ready')
                          : l10n.t('versions.notReady'),
                      style: theme.textTheme.titleSmall,
                    ),
                    const SizedBox(height: Cobalt.space2),
                    for (final g in ar.gaps)
                      Padding(
                        padding: const EdgeInsets.only(bottom: Cobalt.space2),
                        child: Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Padding(
                              padding: const EdgeInsets.only(
                                top: Cobalt.space2,
                              ),
                              child: _GapMark(
                                blocking: g.priority == 'must_clarify_now',
                              ),
                            ),
                            const SizedBox(width: Cobalt.space2),
                            Expanded(
                              child: Text(
                                '${l10n.t('prd.sections.${g.section}')} — ${g.reason}',
                                style: theme.textTheme.bodyMedium,
                              ),
                            ),
                          ],
                        ),
                      ),
                  ],
                  if (commit != null) ...[
                    const SizedBox(height: Cobalt.space3),
                    BlockTag(label: l10n.t('versions.details')),
                    const SizedBox(height: Cobalt.space2),
                    if (commit!.changeSummary.isNotEmpty)
                      Text(
                        commit!.changeSummary,
                        maxLines: 4,
                        overflow: TextOverflow.ellipsis,
                        style: theme.textTheme.bodyLarge,
                      ),
                    Text(
                      l10n.t('versions.diffTitle', {
                        'a': commit!.diffCounts['added'] ?? 0,
                        'c': commit!.diffCounts['changed'] ?? 0,
                        'r': commit!.diffCounts['removed'] ?? 0,
                      }),
                      style: theme.textTheme.bodySmall,
                    ),
                  ],
                ],
              ),
            ),
          ),
          const SizedBox(height: Cobalt.space4),
          Divider(
            height: Cobalt.borderDivider,
            thickness: Cobalt.borderDivider,
            color: palette.strokeSubtle,
          ),
          const SizedBox(height: Cobalt.space4),
          Row(
            children: [
              Expanded(
                child: Text(
                  commit?.committedByName ?? '',
                  style: theme.textTheme.bodySmall,
                ),
              ),
              BlockButton(
                label: l10n.t('prd.backToPrd'),
                variant: BlockButtonVariant.secondary,
                size: BlockButtonSize.sm,
                onPressed: () => context.go('/prd/$prdId'),
              ),
              const SizedBox(width: Cobalt.space2),
              BlockButton(
                key: const Key('commit-now'),
                label: l10n.t('versions.commitNow'),
                icon: Icons.verified_outlined,
                onPressed: onCommit,
              ),
            ],
          ),
        ],
      ),
    );
  }
}

/// Square status mark — the decree replaces the round check/uncheck glyphs.
class _GapMark extends StatelessWidget {
  const _GapMark({required this.blocking});

  final bool blocking;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return Container(
      width: Cobalt.space2,
      height: Cobalt.space2,
      color: blocking ? palette.brand : palette.strokeStrong,
    );
  }
}
