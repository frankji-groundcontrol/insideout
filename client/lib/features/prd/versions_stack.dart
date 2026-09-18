import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import '../../theme/field_card.dart';

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
    return ListView(
      padding: const EdgeInsets.fromLTRB(24, 56, 24, 140),
      children: [
        if (readiness != null)
          Center(
            child: Wrap(
              spacing: 8,
              children: [
                for (final a in const ['validation', 'decision', 'management', 'delivery'])
                  ChoiceChip(label: Text(l10n.t('versions.audience.$a')), selected: a == audience, onSelected: (_) => onAudience(a)),
              ],
            ),
          ),
        const SizedBox(height: 12),
        Text(
          l10n.t('versions.stackEyebrow', {'n': selected?.revision ?? 0, 'total': commits.length}),
          textAlign: TextAlign.center,
          style: theme.textTheme.labelSmall,
        ),
        const SizedBox(height: 20),
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
                      left: 32 + i * 8.0,
                      right: 32 + i * 8.0,
                      child: _PeekCard(name: behind[i].name, onTap: () => onSelect(behind[i].id)),
                    ),
                  Positioned(
                    top: behind.length * 44.0,
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
    );
  }
}

class _PeekCard extends StatelessWidget {
  const _PeekCard({required this.name, required this.onTap});

  final String name;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return GestureDetector(
      onTap: onTap,
      child: Column(
        children: [
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 6),
            decoration: BoxDecoration(
              color: palette.brand,
              borderRadius: const BorderRadius.vertical(top: Radius.circular(10)),
            ),
            child: Text(
              name,
              style: Theme.of(context).textTheme.labelSmall?.copyWith(color: palette.btnFg),
            ),
          ),
          FieldCard(
            color: palette.brandSubtle,
            padding: const EdgeInsets.symmetric(vertical: 18),
            child: const SizedBox(height: 8, width: double.infinity),
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
    final ar = readiness?.audiences[audience];
    return FieldCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.verified, color: Theme.of(context).colorScheme.primary),
              const SizedBox(width: 8),
              if (commit == null)
                Expanded(child: Text(l10n.t('versions.empty'), style: theme.textTheme.headlineSmall))
              else ...[
                Text('${commit!.revision}', style: theme.textTheme.headlineLarge),
                const SizedBox(width: 10),
                Expanded(child: Text(commit!.name, style: theme.textTheme.headlineSmall)),
              ],
            ],
          ),
          const SizedBox(height: 16),
          if (ar != null) ...[
            Text(ar.ready ? l10n.t('versions.ready') : l10n.t('versions.notReady'), style: theme.textTheme.titleSmall),
            const SizedBox(height: 8),
            for (final g in ar.gaps)
              Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Icon(
                      g.priority == 'must_clarify_now' ? Icons.radio_button_unchecked : Icons.check_circle_outline,
                      size: 18,
                      color: Theme.of(context).colorScheme.primary,
                    ),
                    const SizedBox(width: 8),
                    Expanded(child: Text('${l10n.t('prd.sections.${g.section}')} — ${g.reason}', style: theme.textTheme.bodyMedium)),
                  ],
                ),
              ),
          ],
          if (commit != null) ...[
            const SizedBox(height: 12),
            Text(l10n.t('versions.details'), style: theme.textTheme.labelSmall),
            if (commit!.changeSummary.isNotEmpty) Text(commit!.changeSummary, style: theme.textTheme.bodyLarge),
            Text(
              l10n.t('versions.diffTitle', {
                'a': commit!.diffCounts['added'] ?? 0,
                'c': commit!.diffCounts['changed'] ?? 0,
                'r': commit!.diffCounts['removed'] ?? 0,
              }),
              style: theme.textTheme.bodySmall,
            ),
          ],
          const SizedBox(height: 16),
          const Divider(),
          Row(
            children: [
              Expanded(
                child: Text(commit?.committedByName ?? '', style: theme.textTheme.bodySmall),
              ),
              TextButton(onPressed: () => context.go('/prd/$prdId'), child: Text(l10n.t('prd.backToPrd'))),
              const SizedBox(width: 8),
              FilledButton.icon(
                key: const Key('commit-now'),
                onPressed: onCommit,
                icon: const Icon(Icons.verified_outlined, size: 18),
                label: Text(l10n.t('versions.commitNow')),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
