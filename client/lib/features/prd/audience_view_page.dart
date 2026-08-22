import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';

const _audiences = ['decision', 'management', 'delivery', 'validation'];

/// Audience view page: one PRD core projected per reader (PRODUCT.md
/// "One PRD core, multiple audience views"). Rendered from
/// GET /prds/{id}/view — ordered sections with why notes, the
/// audience's readiness gaps, and the latest Commit for context.
/// Read-only by design: the projection never becomes a second document.
class AudienceViewPage extends StatefulWidget {
  const AudienceViewPage({super.key, required this.prdId, this.initialAudience = 'decision'});

  final String prdId;
  final String initialAudience;

  @override
  State<AudienceViewPage> createState() => _AudienceViewPageState();
}

class _AudienceViewPageState extends State<AudienceViewPage> {
  late Future<Map<String, dynamic>> _future;
  late String audience;

  @override
  void initState() {
    super.initState();
    audience = widget.initialAudience;
    _load();
  }

  void _load() {
    _future = context.read<Session>().api.prdView(widget.prdId, audience);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    return AppScaffold(
      title: l10n.t('view.title'),
      actions: [
        Wrap(
          spacing: 6,
          children: _audiences
              .map((a) => ChoiceChip(
                    label: Text(a),
                    selected: a == audience,
                    onSelected: (_) => setState(() {
                      audience = a;
                      _load();
                    }),
                  ))
              .toList(),
        ),
        const SizedBox(width: 12),
      ],
      body: FutureBuilder<Map<String, dynamic>>(
        future: _future,
        builder: (context, snap) {
          if (!snap.hasData) {
            return Center(child: snap.hasError ? Text('${snap.error}') : const CircularProgressIndicator());
          }
          final data = snap.data!;
          final title = data['title'] as String? ?? '';
          final proj = data['projection'] as Map<String, dynamic>;
          final sections = proj['sections'] as Map<String, dynamic>;
          final readiness = data['readiness'] as Map<String, dynamic>;
          final gaps = (readiness['gaps'] as List? ?? const []);
          final latest = data['latestCommit'] as Map<String, dynamic>?;

          return ListView(
            padding: const EdgeInsets.all(16),
            children: [
              Text(title, style: theme.textTheme.headlineSmall),
              const SizedBox(height: 4),
              Text(proj['purpose']?.toString() ?? '', style: theme.textTheme.bodySmall),
              const SizedBox(height: 4),
              Text(l10n.t('view.projectionNote'), style: theme.textTheme.bodySmall),
              if (latest != null) ...[
                const SizedBox(height: 8),
                Text(l10n.t('view.latestCommit', {
                      'n': latest['revision']?.toString() ?? '',
                      'name': latest['name']?.toString() ?? '',
                    }), style: theme.textTheme.bodySmall),
              ],
              const Divider(height: 24),
              for (final key in sections.keys) ...[
                Text(l10n.t('prd.sections.$key'), style: theme.textTheme.titleMedium),
                Text('${l10n.t('view.whyYouRead')}: ${sections[key]}', style: theme.textTheme.bodySmall),
                const SizedBox(height: 4),
                _SectionBody(prdId: widget.prdId, sectionKey: key),
                const SizedBox(height: 16),
              ],
              if (gaps.isNotEmpty) ...[
                const Divider(height: 24),
                Text(l10n.t('view.gapsTitle'), style: theme.textTheme.titleMedium),
                for (final g in gaps)
                  Text('  · ${l10n.t('prd.sections.${g['section']}')} — ${g['reason']}', style: theme.textTheme.bodySmall),
              ],
              const SizedBox(height: 24),
              TextButton(onPressed: () => context.go('/prd/${widget.prdId}/versions'), child: Text(l10n.t('versions.entry'))),
            ],
          );
        },
      ),
    );
  }
}

/// Fetches and renders one section's live content from the PRD itself,
/// so the projection always shows the working version's truth.
class _SectionBody extends StatefulWidget {
  const _SectionBody({required this.prdId, required this.sectionKey});

  final String prdId;
  final String sectionKey;

  @override
  State<_SectionBody> createState() => _SectionBodyState();
}

class _SectionBodyState extends State<_SectionBody> {
  late Future<Prd> _future;

  @override
  void initState() {
    super.initState();
    _future = context.read<Session>().api.getPrd(widget.prdId);
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<Prd>(
      future: _future,
      builder: (context, snap) {
        if (!snap.hasData) return const SizedBox.shrink();
        final content = snap.data!.sections[widget.sectionKey] ?? '';
        if (content.trim().isEmpty) {
          return Text(context.read<Appearance>().t('view.carriedOpen'), style: Theme.of(context).textTheme.bodySmall);
        }
        return Text(content);
      },
    );
  }
}
