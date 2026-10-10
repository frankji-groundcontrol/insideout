import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/block_field.dart';
import '../../theme/block_modal.dart';
import '../../theme/cobalt.dart';

const versionAudiences = ['validation', 'decision', 'management', 'delivery'];

class CommitFormResult {
  CommitFormResult(this.name, this.audience, this.summary, this.note);

  final String name;
  final String audience;
  final String summary;
  final String note;
}

/// The Commit form as a modal **body** on the shared block chrome — it no
/// longer builds its own dialog shell. [showVersionsCommit] hands it to
/// `showBlockModal` and reads [VersionsCommitFormState.result] out of the
/// footer, so the `CommitFormResult` contract is unchanged.
class VersionsCommitForm extends StatefulWidget {
  const VersionsCommitForm({
    super.key,
    required this.initialAudience,
    required this.carryCount,
  });

  final String initialAudience;
  final int carryCount;

  @override
  State<VersionsCommitForm> createState() => VersionsCommitFormState();
}

class VersionsCommitFormState extends State<VersionsCommitForm> {
  late final TextEditingController _name;
  late final TextEditingController _summary;
  late final TextEditingController _note;
  late String _audience;

  @override
  void initState() {
    super.initState();
    _name = TextEditingController();
    _summary = TextEditingController();
    _note = TextEditingController();
    _audience = widget.initialAudience;
  }

  @override
  void dispose() {
    _name.dispose();
    _summary.dispose();
    _note.dispose();
    super.dispose();
  }

  CommitFormResult result() => CommitFormResult(
    _name.text.trim(),
    _audience,
    _summary.text.trim(),
    _note.text.trim(),
  );

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        BlockField(
          label: l10n.t('versions.nameLabel'),
          controller: _name,
          textInputAction: TextInputAction.next,
        ),
        const SizedBox(height: Cobalt.space4),
        BlockSelect<String>(
          label: l10n.t('versions.audienceLabel'),
          value: _audience,
          options: [
            for (final a in versionAudiences)
              (value: a, label: l10n.t('versions.audience.$a')),
          ],
          onChanged: (v) => setState(() => _audience = v ?? _audience),
        ),
        const SizedBox(height: Cobalt.space4),
        BlockTextarea(
          label: l10n.t('versions.summaryLabel'),
          controller: _summary,
          minLines: 2,
          maxLines: 5,
        ),
        const SizedBox(height: Cobalt.space4),
        BlockTextarea(
          key: const Key('commit-note'),
          label: l10n.t('versions.noteLabel'),
          controller: _note,
          minLines: 2,
          maxLines: 5,
        ),
        if (widget.carryCount > 0) ...[
          const SizedBox(height: Cobalt.space4),
          BlockBanner(
            tone: BlockTone.info,
            message: l10n.t('versions.carryInfo', {'n': widget.carryCount}),
          ),
        ],
      ],
    );
  }
}

/// Opens the Commit modal on the one chrome and resolves to the form result,
/// or null when dismissed.
Future<CommitFormResult?> showVersionsCommit({
  required BuildContext context,
  required String initialAudience,
  required int carryCount,
}) {
  final formKey = GlobalKey<VersionsCommitFormState>();
  final t = context.read<Appearance>().t;
  final navigator = Navigator.of(context);
  return showBlockModal<CommitFormResult>(
    context: context,
    title: t('versions.commitTitle'),
    body: VersionsCommitForm(
      key: formKey,
      initialAudience: initialAudience,
      carryCount: carryCount,
    ),
    footer: [
      BlockButton(
        label: t('common.cancel'),
        variant: BlockButtonVariant.secondary,
        onPressed: navigator.pop,
      ),
      BlockButton(
        key: const Key('commit-confirm'),
        label: t('versions.commitNow'),
        icon: Icons.verified_outlined,
        onPressed: () => navigator.pop(formKey.currentState?.result()),
      ),
    ],
  );
}
