import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../chrome/product_dock.dart';
import '../../session/appearance.dart';
import '../../theme/block_button.dart';
import '../../theme/block_chrome.dart';
import '../../theme/block_state.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_theme.dart';
import '../../theme/field_card.dart';

/// The signed-off Home numeral stays giant (DESIGN.md: the Home index block
/// carries a display-scale numeral); the size is a named constant, not a
/// bare literal at the call site.
const double _numeralSize = 120;

/// Clears the top-right utility block on the field.
const double _topInset = 72;

/// The body extends behind the dock (AppScaffold extendBody).
const double _bottomInset = ProductDock.height + Cobalt.space10 + Cobalt.space4;

/// The Home field: loop eyebrow + capture CTA over an index block and the
/// workspace column, centered in a [Cobalt.contentMaxWidth] column (D9).
class HomeField extends StatelessWidget {
  const HomeField({
    super.key,
    required this.workspaces,
    required this.onCreate,
    required this.onJoin,
    required this.onOpen,
    required this.onCapture,
  });

  final List<Workspace> workspaces;
  final VoidCallback onCreate;
  final VoidCallback onJoin;
  final void Function(Workspace w) onOpen;
  final VoidCallback onCapture;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final width = MediaQuery.sizeOf(context).width;
    final wide = width >= Cobalt.bpWide;
    final stacked = width < Cobalt.bpNarrow;
    final hPad = width >= Cobalt.bpDesk
        ? Cobalt.space8
        : wide
        ? Cobalt.space6
        : Cobalt.space4;
    final numeral = workspaces.isEmpty
        ? '00'
        : workspaces.length.toString().padLeft(2, '0');
    final heading = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(l10n.t('home.loopTitle').toUpperCase(), style: eyebrow(context)),
        const SizedBox(height: Cobalt.space2),
        Text(l10n.t('home.loopLead'), style: theme.textTheme.bodyLarge),
      ],
    );
    final capture = BlockButton(
      label: l10n.t('home.captureIdea'),
      icon: Icons.arrow_forward,
      size: BlockButtonSize.lg,
      expand: stacked,
      onPressed: onCapture,
    );
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: Cobalt.contentMaxWidth),
        child: ListView(
          padding: EdgeInsets.fromLTRB(hPad, _topInset, hPad, _bottomInset),
          children: [
            if (stacked)
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  heading,
                  const SizedBox(height: Cobalt.space4),
                  capture,
                ],
              )
            else
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(child: heading),
                  const SizedBox(width: Cobalt.space4),
                  capture,
                ],
              ),
            const SizedBox(height: Cobalt.space8),
            if (wide)
              IntrinsicHeight(
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Expanded(child: _IndexCard(numeral: numeral)),
                    const SizedBox(width: Cobalt.space4),
                    Expanded(
                      child: _WorkspaceColumn(
                        workspaces: workspaces,
                        onOpen: onOpen,
                        onJoin: onJoin,
                        onCreate: onCreate,
                      ),
                    ),
                  ],
                ),
              )
            else ...[
              _IndexCard(numeral: numeral),
              const SizedBox(height: Cobalt.space4),
              _WorkspaceColumn(
                workspaces: workspaces,
                onOpen: onOpen,
                onJoin: onJoin,
                onCreate: onCreate,
              ),
            ],
            const SizedBox(height: Cobalt.space6),
            Text(l10n.t('home.loopTrail'), style: eyebrow(context)),
          ],
        ),
      ),
    );
  }
}

class _IndexCard extends StatelessWidget {
  const _IndexCard({required this.numeral});

  final String numeral;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    return FieldCard(
      color: palette.surfaceSunken,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.t('home.chapterEyebrow', {'n': numeral}),
            style: eyebrow(context),
          ),
          const SizedBox(height: Cobalt.space3),
          Text(
            l10n.t('home.chapterTitle'),
            style: theme.textTheme.headlineLarge,
          ),
          const SizedBox(height: Cobalt.space2),
          Text(
            numeral,
            style: theme.textTheme.displayLarge?.copyWith(
              color: palette.brand,
              fontSize: _numeralSize,
              height: 0.9,
            ),
          ),
          const SizedBox(height: Cobalt.space3),
          Text(l10n.t('home.chapterBody'), style: theme.textTheme.bodyLarge),
        ],
      ),
    );
  }
}

class _WorkspaceColumn extends StatelessWidget {
  const _WorkspaceColumn({
    required this.workspaces,
    required this.onOpen,
    required this.onJoin,
    required this.onCreate,
  });

  final List<Workspace> workspaces;
  final void Function(Workspace w) onOpen;
  final VoidCallback onJoin;
  final VoidCallback onCreate;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    if (workspaces.isEmpty) {
      return BlockEmpty(
        icon: Icons.workspaces_outlined,
        title: l10n.t('workspace.title'),
        body: l10n.t('workspace.empty'),
        action: Wrap(
          spacing: Cobalt.space3,
          runSpacing: Cobalt.space3,
          alignment: WrapAlignment.center,
          children: [
            BlockButton(
              label: l10n.t('workspace.create'),
              icon: Icons.add,
              onPressed: onCreate,
            ),
            BlockButton(
              label: l10n.t('workspace.join'),
              variant: BlockButtonVariant.secondary,
              onPressed: onJoin,
            ),
          ],
        ),
      );
    }
    return Column(
      children: [
        FieldCard(
          onTap: onJoin,
          padding: const EdgeInsets.all(Cobalt.space4),
          child: Row(
            children: [
              const BlockIconWell(
                icon: Icons.group_add_outlined,
                size: 40,
                iconSize: 20,
              ),
              const SizedBox(width: Cobalt.space3),
              Expanded(
                child: Text(
                  l10n.t('workspace.joinTitle'),
                  style: theme.textTheme.titleMedium,
                ),
              ),
              Icon(Icons.chevron_right, color: palette.fgMuted),
            ],
          ),
        ),
        for (var i = 0; i < workspaces.length; i++) ...[
          const SizedBox(height: Cobalt.space3),
          FieldCard(
            onTap: () => onOpen(workspaces[i]),
            padding: const EdgeInsets.all(Cobalt.space4),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Text(
                      (i + 1).toString().padLeft(2, '0'),
                      style: theme.textTheme.displaySmall?.copyWith(
                        fontSize: 48,
                        height: 1,
                      ),
                    ),
                    const Spacer(),
                    Icon(Icons.chevron_right, color: palette.fgMuted),
                  ],
                ),
                const SizedBox(height: Cobalt.space3),
                Text(workspaces[i].title, style: theme.textTheme.titleLarge),
                const SizedBox(height: Cobalt.space2),
                Text(
                  '${l10n.t('workspace.role.${workspaces[i].myRole}')} · ${l10n.t('workspace.members', {'count': workspaces[i].memberCount})}',
                  style: theme.textTheme.bodySmall,
                ),
              ],
            ),
          ),
        ],
      ],
    );
  }
}
