import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';

const _audiences = ['validation', 'decision', 'management', 'delivery'];

/// Versions page: the human Commit list plus the "form a version now"
/// readiness disclosure (docs/plans/2026-08-21-version-commit.md).
class VersionsPage extends StatefulWidget {
  const VersionsPage({super.key, required this.prdId});

  final String prdId;

  @override
  State<VersionsPage> createState() => _VersionsPageState();
}

class _VersionsPageState extends State<VersionsPage> {
  late Future<List<PrdCommit>> _commits;
  late Future<PrdReadiness> _readiness;
  String audience = 'decision';
  String? selectedId;

  @override
  void initState() {
    super.initState();
    _reload();
  }

  void _reload() {
    final api = context.read<Session>().api;
    _commits = api.listPrdCommits(widget.prdId);
    _readiness = api.prdReadiness(widget.prdId);
  }

  Future<void> _commitDialog() async {
    List<String> carry;
    try {
      carry = (await _readiness).audiences[audience]?.carryIntoCommit ?? const [];
    } catch (_) {
      carry = const [];
    }
    if (!mounted) return;
    final result = await showDialog<_CommitResult>(
      context: context,
      builder: (ctx) => _CommitDialog(initialAudience: audience, carryCount: carry.length),
    );
    if (result == null || result.name.isEmpty) return;
    await context.read<Session>().api.commitPrd(
          widget.prdId,
          name: result.name,
          audience: result.audience,
          summary: result.summary,
          unresolved: carry,
        );
    if (mounted) setState(_reload);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.read<Appearance>();
    return AppScaffold(
      title: l10n.t('versions.title'),
      fab: FloatingActionButton.extended(
        onPressed: _commitDialog,
        icon: const Icon(Icons.verified_outlined),
        label: Text(l10n.t('versions.commitNow')),
      ),
      body: FutureBuilder(
        future: _readiness,
        builder: (context, rsnap) {
          return Row(
            children: [
              SizedBox(
                width: 300,
                child: Column(
                  children: [
                    Padding(
                      padding: const EdgeInsets.all(12),
                      child: rsnap.hasData
                          ? Wrap(
                              spacing: 6,
                              children: _audiences
                                  .map((a) => ChoiceChip(
                                        label: Text(a),
                                        selected: a == audience,
                                        onSelected: (_) => setState(() => audience = a),
                                      ))
                                  .toList(),
                            )
                          : const SizedBox.shrink(),
                    ),
                    Expanded(
                      child: rsnap.hasData
                          ? _gapList(l10n, rsnap.data!)
                          : rsnap.hasError
                              ? Center(child: Text('${rsnap.error}'))
                              : const Center(child: CircularProgressIndicator()),
                    ),
                  ],
                ),
              ),
              const VerticalDivider(width: 1),
              Expanded(
                child: FutureBuilder(
                  future: _commits,
                  builder: (context, snap) {
                    if (!snap.hasData) {
                      return Center(child: snap.hasError ? Text('${snap.error}') : const CircularProgressIndicator());
                    }
                    if (snap.data!.isEmpty) {
                      return Center(child: Text(l10n.t('versions.empty')));
                    }
                    final selected =
                        snap.data!.cast<PrdCommit?>().firstWhere((c) => c!.id == selectedId, orElse: () => snap.data!.first)!;
                    return Row(
                      children: [
                        SizedBox(
                          width: 280,
                          child: ListView(
                            children: snap.data!
                                .map((c) => ListTile(
                                      selected: c.id == selected.id,
                                      title: Text(c.name),
                                      subtitle: Text(
                                        '${l10n.t('versions.revisionN', {'n': c.revision})} · ${c.primaryAudience} · ${c.createdAt.substring(0, 10)}',
                                      ),
                                      onTap: () => setState(() => selectedId = c.id),
                                    ))
                                .toList(),
                          ),
                        ),
                        const VerticalDivider(width: 1),
                        Expanded(child: _commitDetail(l10n, selected)),
                      ],
                    );
                  },
                ),
              ),
            ],
          );
        },
      ),
    );
  }

  Widget _gapList(Appearance l10n, PrdReadiness r) {
    final ar = r.audiences[audience];
    if (ar == null) return const SizedBox.shrink();
    return ListView(
      padding: const EdgeInsets.all(12),
      children: [
        Text(ar.ready ? l10n.t('versions.ready') : l10n.t('versions.notReady'),
            style: Theme.of(context).textTheme.titleSmall),
        const SizedBox(height: 8),
        for (final g in ar.gaps)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Icon(
                  g.priority == 'must_clarify_now'
                      ? Icons.error_outline
                      : (g.priority == 'should_clarify_this_version' ? Icons.help_outline : Icons.schedule),
                  size: 18,
                  color: g.priority == 'must_clarify_now' ? Theme.of(context).colorScheme.error : null,
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Text('${l10n.t('prd.sections.${g.section}')} — ${g.reason}',
                      style: Theme.of(context).textTheme.bodySmall),
                ),
              ],
            ),
          ),
      ],
    );
  }

  Widget _commitDetail(Appearance l10n, PrdCommit c) {
    final theme = Theme.of(context);
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text(c.name, style: theme.textTheme.headlineSmall),
        const SizedBox(height: 4),
        Text(
          '${l10n.t('versions.revisionN', {'n': c.revision})} · ${c.primaryAudience}'
          '${c.committedByName?.isNotEmpty == true ? ' · ${c.committedByName}' : ''}',
          style: theme.textTheme.bodySmall,
        ),
        if (c.changeSummary.isNotEmpty) ...[
          const SizedBox(height: 12),
          Text(c.changeSummary),
        ],
        const SizedBox(height: 16),
        Text(
          l10n.t('versions.diffTitle', {'a': c.diffCounts['added'] ?? 0, 'c': c.diffCounts['changed'] ?? 0, 'r': c.diffCounts['removed'] ?? 0}),
          style: theme.textTheme.titleSmall,
        ),
        for (final e in c.diffSections.entries)
          Text('  ${l10n.t('prd.sections.${e.key}')}: ${l10n.t('versions.diff.${e.value}')}', style: theme.textTheme.bodySmall),
        if (c.unresolved.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text(l10n.t('versions.unresolvedTitle'), style: theme.textTheme.titleSmall),
          for (final u in c.unresolved) Text('  · $u', style: theme.textTheme.bodySmall),
        ],
        if (c.decisionNote.isNotEmpty) ...[
          const SizedBox(height: 16),
          Text(l10n.t('versions.noteTitle'), style: theme.textTheme.titleSmall),
          Text(c.decisionNote, style: theme.textTheme.bodySmall),
        ],
        const SizedBox(height: 24),
        TextButton(onPressed: () => context.go('/prd/${widget.prdId}'), child: Text(l10n.t('prd.backToPrd'))),
      ],
    );
  }
}

class _CommitResult {
  _CommitResult(this.name, this.audience, this.summary);

  final String name;
  final String audience;
  final String summary;
}

class _CommitDialog extends StatefulWidget {
  const _CommitDialog({required this.initialAudience, required this.carryCount});

  final String initialAudience;
  final int carryCount;

  @override
  State<_CommitDialog> createState() => _CommitDialogState();
}

class _CommitDialogState extends State<_CommitDialog> {
  late final TextEditingController _name;
  late final TextEditingController _summary;
  late String _audience;

  @override
  void initState() {
    super.initState();
    _name = TextEditingController();
    _summary = TextEditingController();
    _audience = widget.initialAudience;
  }

  @override
  void dispose() {
    _name.dispose();
    _summary.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.read<Appearance>();
    return AlertDialog(
      title: Text(l10n.t('versions.commitTitle')),
      content: SizedBox(
        width: 420,
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(controller: _name, decoration: InputDecoration(labelText: l10n.t('versions.nameLabel'))),
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                initialValue: _audience,
                decoration: InputDecoration(labelText: l10n.t('versions.audienceLabel')),
                items: _audiences.map((a) => DropdownMenuItem(value: a, child: Text(a))).toList(),
                onChanged: (v) => setState(() => _audience = v ?? _audience),
              ),
              const SizedBox(height: 12),
              TextField(controller: _summary, decoration: InputDecoration(labelText: l10n.t('versions.summaryLabel'))),
              if (widget.carryCount > 0) ...[
                const SizedBox(height: 12),
                Text(l10n.t('versions.carryInfo', {'n': widget.carryCount}), style: Theme.of(context).textTheme.bodySmall),
              ],
            ],
          ),
        ),
      ),
      actions: [
        TextButton(onPressed: () => Navigator.pop(context), child: Text(l10n.t('common.cancel'))),
        FilledButton(
          onPressed: () => Navigator.pop(context, _CommitResult(_name.text.trim(), _audience, _summary.text.trim())),
          child: Text(l10n.t('versions.commitNow')),
        ),
      ],
    );
  }
}
