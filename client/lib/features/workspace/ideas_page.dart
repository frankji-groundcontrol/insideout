import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/loop_focus.dart';
import '../../chrome/product_dock.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_field.dart';
import '../../theme/block_form.dart';
import '../../theme/block_modal.dart';
import '../../theme/block_state.dart';
import '../../theme/cobalt.dart';
import '../../widgets/guarded_action.dart';
import 'idea_poster.dart';

/// The wall keeps breathing room under the top-right utility block and above
/// the dock (AppScaffold extendBody — the body scrolls behind it).
const double _topInset = 56;
const double _bottomInset = ProductDock.height + Cobalt.space10 + Cobalt.space4;

/// The capture bar reads as one instrument: capped narrower than the content
/// column so the eye lands mid-page.
const double _captureMaxWidth = 720;

class IdeasPage extends StatefulWidget {
  const IdeasPage({super.key, required this.workspaceId});

  final String workspaceId;

  @override
  State<IdeasPage> createState() => _IdeasPageState();
}

class _IdeasPageState extends State<IdeasPage> {
  late Future<List<Idea>> _future;
  final _capture = TextEditingController();

  @override
  void initState() {
    super.initState();
    context.read<LoopFocus>().rememberWorkspace(widget.workspaceId);
    _future = context.read<Session>().api.listIdeas(widget.workspaceId);
    _seedProject();
  }

  Future<void> _seedProject() async {
    final focus = context.read<LoopFocus>();
    if (focus.projectFor(widget.workspaceId) != null) return;
    try {
      final projects = await context.read<Session>().api.listProjects(
        widget.workspaceId,
      );
      if (!mounted ||
          focus.projectFor(widget.workspaceId) != null ||
          projects.isEmpty) {
        return;
      }
      focus.rememberProject(projects.first.id, workspaceId: widget.workspaceId);
    } catch (_) {}
  }

  @override
  void dispose() {
    _capture.dispose();
    super.dispose();
  }

  Future<void> _reload() async {
    // Block body on purpose: `() => _future = …` returns the Future, which
    // the framework flags ("setState() callback argument returned a Future").
    setState(() {
      _future = context.read<Session>().api.listIdeas(widget.workspaceId);
    });
  }

  Future<void> _captureNow() async {
    final title = _capture.text.trim();
    if (title.isEmpty) return;
    // 4.2: on failure the SnackBar shows and the typed text stays in the
    // field for a retry — only a successful write clears it.
    final ok = await guardWrite(
      context,
      () =>
          context.read<Session>().api.createIdea(widget.workspaceId, title, ''),
    );
    if (!ok || !mounted) return;
    _capture.clear();
    await _reload();
  }

  // USER-09: drop is a terminal action on a one-tap menu item — confirm
  // with the translated prompt before the card disappears for good.
  Future<void> _drop(Idea idea) async {
    final t = context.read<Appearance>().t;
    final ok = await showBlockConfirm(
      context: context,
      title: t('common.confirmTitle'),
      message: t('idea.dropConfirm'),
      confirmText: t('idea.drop'),
      danger: true,
    );
    if (!ok || !mounted) return;
    final done = await guardWrite(
      context,
      () => context.read<Session>().api.dropIdea(idea.id),
    );
    if (done && mounted) await _reload();
  }

  Future<void> _edit(Idea idea) async {
    final t = context.read<Appearance>().t;
    final values = await showBlockForm(
      context: context,
      title: t('common.edit'),
      confirmLabel: t('common.save'),
      fields: [
        BlockFormField(label: t('idea.titlePlaceholder'), initial: idea.title),
        BlockFormField(
          label: t('idea.contentPlaceholder'),
          initial: idea.content,
          multiline: true,
        ),
      ],
    );
    if (values == null || !mounted) return;
    final done = await guardWrite(
      context,
      () => context.read<Session>().api.updateIdea(
        idea.id,
        title: values[0].trim(),
        content: values[1],
      ),
    );
    if (done && mounted) await _reload();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return AppScaffold(
      body: FutureBuilder<List<Idea>>(
        future: _future,
        builder: (context, snap) {
          if (snap.hasError) {
            return BlockError(error: snap.error, onRetry: _reload);
          }
          if (!snap.hasData) return const BlockLoading();
          return _IdeasWall(
            ideas: snap.data!,
            capture: _capture,
            onCapture: _captureNow,
            onEdit: _edit,
            onDrop: _drop,
            onConvert: (idea) async {
              // 4.2: on failure the guard shows the error and the idea card
              // stays as-is — nothing is cleared, so retrying is safe.
              await guardWrite(context, () async {
                final ids = await context.read<Session>().api.convertIdea(
                  idea.id,
                );
                final prdId = ids['prdId'];
                if (prdId != null && context.mounted) {
                  context.read<LoopFocus>().rememberPrd(prdId);
                  context.go('/prd/$prdId');
                }
              });
            },
            onOpenPrd: (idea) {
              if (idea.prdId == null) return;
              context.read<LoopFocus>().rememberPrd(idea.prdId!);
              context.go('/prd/${idea.prdId}');
            },
            emptyLabel: l10n.t('idea.empty'),
          );
        },
      ),
    );
  }
}

class _IdeasWall extends StatelessWidget {
  const _IdeasWall({
    required this.ideas,
    required this.capture,
    required this.onCapture,
    required this.onEdit,
    required this.onDrop,
    required this.onConvert,
    required this.onOpenPrd,
    required this.emptyLabel,
  });

  final List<Idea> ideas;
  final TextEditingController capture;
  final VoidCallback onCapture;
  final void Function(Idea idea) onEdit;
  final void Function(Idea idea) onDrop;
  final void Function(Idea idea) onConvert;
  final void Function(Idea idea) onOpenPrd;
  final String emptyLabel;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    final width = MediaQuery.sizeOf(context).width;
    final display = Cobalt.ideasDisplaySize(width);
    final hPad = width >= Cobalt.bpDesk
        ? Cobalt.space8
        : width >= Cobalt.bpNarrow
        ? Cobalt.space6
        : Cobalt.space4;
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: Cobalt.contentMaxWidth),
        child: ListView(
          padding: EdgeInsets.fromLTRB(hPad, _topInset, hPad, _bottomInset),
          children: [
            Text(
              l10n.t('idea.wallTitle').toUpperCase(),
              textAlign: TextAlign.center,
              style: theme.textTheme.displayLarge?.copyWith(
                fontSize: display,
                height: 0.9,
                color: palette.brand,
              ),
            ),
            const SizedBox(height: Cobalt.space8),
            Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: _captureMaxWidth),
                child: Row(
                  children: [
                    Expanded(
                      child: BlockField(
                        controller: capture,
                        hint: l10n.t('idea.captureHint'),
                        onSubmitted: (_) => onCapture(),
                      ),
                    ),
                    const SizedBox(width: Cobalt.space3),
                    BlockButton(
                      label: l10n.t('idea.capture'),
                      onPressed: onCapture,
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: Cobalt.space3),
            Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: _captureMaxWidth),
                child: Text(
                  l10n.t('idea.privateHint'),
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: palette.fgMuted,
                  ),
                  textAlign: TextAlign.center,
                ),
              ),
            ),
            const SizedBox(height: Cobalt.space8),
            if (ideas.isEmpty)
              BlockEmpty(icon: Icons.lightbulb_outline, body: emptyLabel)
            else ...[
              LayoutBuilder(
                builder: (context, constraints) {
                  final w = constraints.maxWidth;
                  final cols = w >= Cobalt.bpWide
                      ? 3
                      : w >= Cobalt.bpNarrow
                      ? 2
                      : 1;
                  const gap = Cobalt.space4;
                  final cardWidth = (w - gap * (cols - 1)) / cols;
                  return Wrap(
                    spacing: gap,
                    runSpacing: gap,
                    children: [
                      for (final idea in ideas)
                        SizedBox(
                          width: cardWidth,
                          child: IdeaPoster(
                            idea: idea,
                            onEdit: () => onEdit(idea),
                            onDrop: () => onDrop(idea),
                            onConvert: () => onConvert(idea),
                            onOpenPrd: () => onOpenPrd(idea),
                          ),
                        ),
                    ],
                  );
                },
              ),
              const SizedBox(height: Cobalt.space6),
              Text(
                l10n.t('idea.noFolders'),
                style: theme.textTheme.labelSmall?.copyWith(letterSpacing: 0.8),
                textAlign: TextAlign.center,
              ),
            ],
          ],
        ),
      ),
    );
  }
}
