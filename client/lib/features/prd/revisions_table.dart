import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/cobalt.dart';

class RevisionsTable extends StatelessWidget {
  const RevisionsTable({
    super.key,
    required this.revisions,
    required this.selected,
    required this.onSelect,
    required this.onSnapshot,
  });

  final List<PrdRevision> revisions;
  final PrdRevision selected;
  final ValueChanged<String> onSelect;
  final VoidCallback onSnapshot;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final palette = Cobalt.of(context);
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: Cobalt.contentMaxWidth),
        child: Padding(
          padding: const EdgeInsets.fromLTRB(
            Cobalt.space6,
            Cobalt.space4,
            Cobalt.space6,
            140,
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Align(
                      alignment: Alignment.centerLeft,
                      child: BlockTag(label: l10n.t('prd.snapshotLabel')),
                    ),
                  ),
                  BlockButton(
                    key: const Key('revisions-snapshot'),
                    label: l10n.t('prd.snapshotRevision'),
                    icon: Icons.add,
                    size: BlockButtonSize.sm,
                    onPressed: onSnapshot,
                  ),
                ],
              ),
              const SizedBox(height: Cobalt.space3),
              Divider(
                height: Cobalt.borderContainer,
                thickness: Cobalt.borderContainer,
                color: palette.brand,
              ),
              const SizedBox(height: Cobalt.space3),
              Expanded(
                child: LayoutBuilder(
                  builder: (context, box) {
                    final rows = [
                      for (final r in revisions)
                        _RevisionRow(
                          revision: r,
                          selected: r.id == selected.id,
                          onTap: () => onSelect(r.id),
                        ),
                    ];
                    final detail = _detailChildren(context, selected);
                    // Below the narrow breakpoint a 320px sidebar would leave
                    // the detail column a sliver, so the table stacks.
                    if (box.maxWidth < Cobalt.bpNarrow) {
                      return ListView(
                        padding: const EdgeInsets.only(bottom: Cobalt.space6),
                        children: [
                          ...rows,
                          const SizedBox(height: Cobalt.space4),
                          Divider(
                            height: Cobalt.borderContainer,
                            thickness: Cobalt.borderContainer,
                            color: palette.brand,
                          ),
                          const SizedBox(height: Cobalt.space4),
                          ...detail,
                        ],
                      );
                    }
                    return Row(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        SizedBox(width: 320, child: ListView(children: rows)),
                        VerticalDivider(
                          width: Cobalt.borderDivider,
                          thickness: Cobalt.borderDivider,
                          color: palette.strokeSubtle,
                        ),
                        Expanded(
                          child: ListView(
                            padding: const EdgeInsets.fromLTRB(
                              Cobalt.space6,
                              0,
                              Cobalt.space6,
                              Cobalt.space6,
                            ),
                            children: detail,
                          ),
                        ),
                      ],
                    );
                  },
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// One row of the snapshot table — hover tints, selection carries a 4px
/// cobalt edge. Flat by design: a stack of cast blocks would read as cards,
/// not as a table.
class _RevisionRow extends StatefulWidget {
  const _RevisionRow({
    required this.revision,
    required this.selected,
    required this.onTap,
  });

  final PrdRevision revision;
  final bool selected;
  final VoidCallback onTap;

  @override
  State<_RevisionRow> createState() => _RevisionRowState();
}

class _RevisionRowState extends State<_RevisionRow> {
  var hover = false;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    final r = widget.revision;
    final bg = widget.selected
        ? palette.brandSubtle
        : hover
        ? palette.surfaceSunken
        : Colors.transparent;
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => hover = true),
      onExit: (_) => setState(() => hover = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          key: const Key('revision-row'),
          padding: const EdgeInsets.symmetric(
            horizontal: Cobalt.space3,
            vertical: Cobalt.space3,
          ),
          decoration: BoxDecoration(
            color: bg,
            // The 4px left edge is always laid out (transparent when not
            // selected) so picking a row never shifts the table.
            border: Border(
              left: BorderSide(
                color: widget.selected ? palette.brand : Colors.transparent,
                width: Cobalt.space1,
              ),
              bottom: BorderSide(
                color: palette.strokeSubtle,
                width: Cobalt.borderDivider,
              ),
            ),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                l10n.t('prd.revisionN', {'n': r.revision}),
                style: theme.textTheme.titleSmall,
              ),
              const SizedBox(height: Cobalt.space1),
              Text(
                r.note?.isNotEmpty == true ? r.note! : r.createdAt,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.bodySmall,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// The selected snapshot's section bodies, shared by the wide split (its own
/// scroll column) and the narrow stack (appended under the row list).
List<Widget> _detailChildren(BuildContext context, PrdRevision revision) {
  final l10n = context.watch<Appearance>();
  final theme = Theme.of(context);
  final palette = Cobalt.of(context);
  return [
    for (var i = 0; i < prdSectionKeys.length; i++) ...[
      Text(
        l10n.t('prd.sections.${prdSectionKeys[i]}'),
        style: theme.textTheme.titleSmall,
      ),
      const SizedBox(height: Cobalt.space1),
      Text(
        revision.sections[prdSectionKeys[i]]?.isNotEmpty == true
            ? revision.sections[prdSectionKeys[i]]!
            : '—',
        style: theme.textTheme.bodyMedium,
      ),
      if (i < prdSectionKeys.length - 1) ...[
        const SizedBox(height: Cobalt.space4),
        Divider(
          height: Cobalt.borderDivider,
          thickness: Cobalt.borderDivider,
          color: palette.strokeSubtle,
        ),
        const SizedBox(height: Cobalt.space4),
      ],
    ],
  ];
}
