import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/product_dock.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_state.dart';
import '../../theme/cobalt.dart';
import '../../theme/field_card.dart';
import 'audience_selector.dart';

const _audiences = ['decision', 'management', 'delivery', 'validation'];

/// Audience view page: one PRD core projected per reader (PRODUCT.md
/// "One PRD core, multiple audience views"). Rendered from
/// GET /prds/{id}/view — ordered sections with why notes, the
/// audience's readiness gaps, and the latest Commit for context.
/// Read-only by design: the projection never becomes a second document.
class AudienceViewPage extends StatefulWidget {
  const AudienceViewPage({
    super.key,
    required this.prdId,
    this.initialAudience = 'decision',
  });

  final String prdId;
  final String initialAudience;

  @override
  State<AudienceViewPage> createState() => _AudienceViewPageState();
}

class _AudienceViewPageState extends State<AudienceViewPage> {
  late Future<(Map<String, dynamic>, Prd)> _future;
  late String audience;

  @override
  void initState() {
    super.initState();
    audience = widget.initialAudience;
    _load();
  }

  // The view carries the reading order and whys; the section text is the PRD's
  // own, fetched once here for every card instead of once per card.
  void _load() {
    final api = context.read<Session>().api;
    _future = (
      api.prdView(widget.prdId, audience),
      api.getPrd(widget.prdId),
    ).wait;
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return AppScaffold(
      title: l10n.t('view.title'),
      paintTitle: true,
      body: FutureBuilder<(Map<String, dynamic>, Prd)>(
        future: _future,
        builder: (context, snap) {
          if (snap.hasError) {
            return BlockError(
              error: snap.error,
              onRetry: () => setState(_load),
            );
          }
          if (!snap.hasData) return const BlockLoading();
          final (view, prd) = snap.data!;
          return _Projection(
            prdId: widget.prdId,
            data: view,
            prd: prd,
            audience: audience,
            onAudience: (a) => setState(() {
              audience = a;
              _load();
            }),
          );
        },
      ),
    );
  }
}

class _Projection extends StatelessWidget {
  const _Projection({
    required this.prdId,
    required this.data,
    required this.prd,
    required this.audience,
    required this.onAudience,
  });

  final String prdId;
  final Map<String, dynamic> data;
  final Prd prd;
  final String audience;
  final ValueChanged<String> onAudience;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    final title = data['title'] as String? ?? '';
    final proj = data['projection'] as Map<String, dynamic>;
    // The server sends the reading order as a list of {section, why}.
    final picks = (proj['sections'] as List).cast<Map<String, dynamic>>();
    final readiness = data['readiness'] as Map<String, dynamic>;
    final gaps = (readiness['gaps'] as List? ?? const []);
    final blocking = [
      for (final g in gaps)
        if (g is Map && g['priority']?.toString() == 'must_clarify_now') g,
    ];
    final later = [
      for (final g in gaps)
        if (g is! Map || g['priority']?.toString() != 'must_clarify_now') g,
    ];
    final latest = data['latestCommit'] as Map<String, dynamic>?;

    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: Cobalt.contentMaxWidth),
        child: ListView(
          padding: const EdgeInsets.fromLTRB(
            Cobalt.space6,
            Cobalt.space4,
            Cobalt.space6,
            ProductDock.clearance,
          ),
          children: [
            AudienceSelector(
              audiences: _audiences,
              selected: audience,
              onSelected: onAudience,
              labels: (a) => l10n.t('versions.audience.$a'),
              alignment: WrapAlignment.start,
            ),
            const SizedBox(height: Cobalt.space6),
            Text(title, style: theme.textTheme.headlineSmall),
            const SizedBox(height: Cobalt.space2),
            Text(
              proj['purpose']?.toString() ?? '',
              style: theme.textTheme.bodyMedium,
            ),
            const SizedBox(height: Cobalt.space2),
            Text(
              l10n.t('view.projectionNote'),
              style: theme.textTheme.bodySmall,
            ),
            if (latest != null)
              Container(
                width: double.infinity,
                margin: const EdgeInsets.only(top: Cobalt.space3),
                padding: const EdgeInsets.symmetric(
                  horizontal: Cobalt.space3,
                  vertical: Cobalt.space2,
                ),
                decoration: BoxDecoration(
                  color: palette.surfaceSunken,
                  border: Border(
                    left: BorderSide(color: palette.brand, width: 4),
                  ),
                ),
                child: Text(
                  l10n.t('view.latestCommit', {
                    'n': latest['revision']?.toString() ?? '',
                    'name': latest['name']?.toString() ?? '',
                  }),
                  style: theme.textTheme.bodySmall,
                ),
              ),
            const SizedBox(height: Cobalt.space6),
            for (final pick in picks)
              Padding(
                padding: const EdgeInsets.only(bottom: Cobalt.space4),
                child: FieldCard(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        l10n.t('prd.sections.${pick['section']}'),
                        style: theme.textTheme.titleLarge,
                      ),
                      const SizedBox(height: Cobalt.space2),
                      Text(
                        '${l10n.t('view.whyYouRead')}: ${pick['why']}',
                        style: theme.textTheme.bodySmall,
                      ),
                      const SizedBox(height: Cobalt.space3),
                      Divider(
                        height: Cobalt.borderDivider,
                        thickness: Cobalt.borderDivider,
                        color: palette.strokeSubtle,
                      ),
                      const SizedBox(height: Cobalt.space3),
                      _SectionBody(
                        content: prd.sections[pick['section']] ?? '',
                      ),
                    ],
                  ),
                ),
              ),
            if (gaps.isNotEmpty) ...[
              const SizedBox(height: Cobalt.space4),
              Container(
                padding: const EdgeInsets.only(bottom: Cobalt.space2),
                decoration: BoxDecoration(
                  border: Border(
                    bottom: BorderSide(
                      color: palette.brand,
                      width: Cobalt.borderContainer,
                    ),
                  ),
                ),
                child: Text(
                  l10n.t('view.gapsTitle'),
                  style: theme.textTheme.titleMedium,
                ),
              ),
              const SizedBox(height: Cobalt.space3),
              for (final g in blocking) _GapRow(gap: g, blocking: true),
              if (later.isNotEmpty) ...[
                const SizedBox(height: Cobalt.space4),
                Text(
                  l10n.t('roadmap.bandLater'),
                  style: theme.textTheme.titleSmall?.copyWith(
                    letterSpacing: 0.6,
                  ),
                ),
                const SizedBox(height: Cobalt.space2),
                for (final g in later) _GapRow(gap: g, blocking: false),
              ],
            ],
            const SizedBox(height: Cobalt.space6),
            Align(
              alignment: Alignment.centerLeft,
              child: BlockButton(
                label: l10n.t('versions.entry'),
                icon: Icons.layers_outlined,
                variant: BlockButtonVariant.secondary,
                onPressed: () => context.go('/prd/$prdId/versions'),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// One readiness gap: a square mark (never a dot) plus the localized section
/// name and the reason. The raw priority enum never reaches the page.
class _GapRow extends StatelessWidget {
  const _GapRow({required this.gap, required this.blocking});

  final Map<dynamic, dynamic> gap;
  final bool blocking;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: Cobalt.space2),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: Cobalt.space2,
            height: Cobalt.space2,
            margin: const EdgeInsets.only(top: Cobalt.space2),
            color: blocking ? palette.brand : palette.strokeStrong,
          ),
          const SizedBox(width: Cobalt.space2),
          Expanded(
            child: Text(
              '${l10n.t('prd.sections.${gap['section']}')} — ${gap['reason']}',
              style: theme.textTheme.bodyMedium,
            ),
          ),
        ],
      ),
    );
  }
}

/// One section's live content from the PRD itself, so the projection always
/// shows the working version's truth.
class _SectionBody extends StatelessWidget {
  const _SectionBody({required this.content});

  final String content;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (content.trim().isEmpty) {
      return Text(
        context.read<Appearance>().t('view.carriedOpen'),
        style: theme.textTheme.bodySmall,
      );
    }
    return Text(content, style: theme.textTheme.bodyMedium);
  }
}
