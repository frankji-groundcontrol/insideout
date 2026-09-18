import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';

const versionAudiences = ['validation', 'decision', 'management', 'delivery'];

class CommitFormResult {
  CommitFormResult(this.name, this.audience, this.summary);

  final String name;
  final String audience;
  final String summary;
}

class VersionsCommitDialog extends StatefulWidget {
  const VersionsCommitDialog({super.key, required this.initialAudience, required this.carryCount});

  final String initialAudience;
  final int carryCount;

  @override
  State<VersionsCommitDialog> createState() => _VersionsCommitDialogState();
}

class _VersionsCommitDialogState extends State<VersionsCommitDialog> {
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
                items: versionAudiences.map((a) => DropdownMenuItem(value: a, child: Text(a))).toList(),
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
          key: const Key('commit-confirm'),
          onPressed: () => Navigator.pop(context, CommitFormResult(_name.text.trim(), _audience, _summary.text.trim())),
          child: Text(l10n.t('versions.commitNow')),
        ),
      ],
    );
  }
}
