import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../api/errors_l10n.dart';
import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/loop_focus.dart';
import '../../chrome/time_ago.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/cobalt.dart';
import '../../theme/field_card.dart';

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
    if (focus.projectId != null) return;
    try {
      final projects = await context.read<Session>().api.listProjects(widget.workspaceId);
      if (!mounted || focus.projectId != null || projects.isEmpty) return;
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
    await context.read<Session>().api.createIdea(widget.workspaceId, title, '');
    _capture.clear();
    await _reload();
  }

  // USER-09: drop is a terminal action on a one-tap menu item — confirm
  // with the translated prompt before the card disappears for good.
  Future<void> _drop(Idea idea) async {
    final l10n = context.read<Appearance>();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.t('idea.dropConfirm')),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('idea.drop'))),
        ],
      ),
    );
    if (ok == true && mounted) {
      await context.read<Session>().api.dropIdea(idea.id);
      await _reload();
    }
  }

  Future<void> _edit(Idea idea) async {
    final l10n = context.read<Appearance>();
    final title = TextEditingController(text: idea.title);
    final content = TextEditingController(text: idea.content);
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.t('common.edit')),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(controller: title, decoration: InputDecoration(labelText: l10n.t('idea.titlePlaceholder'))),
            TextField(controller: content, decoration: InputDecoration(labelText: l10n.t('idea.contentPlaceholder')), maxLines: 4),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(l10n.t('common.cancel'))),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(l10n.t('common.save'))),
        ],
      ),
    );
    if (ok == true && mounted) {
      await context.read<Session>().api.updateIdea(idea.id, title: title.text.trim(), content: content.text);
      await _reload();
    }
    title.dispose();
    content.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    return AppScaffold(
      body: FutureBuilder(
        future: _future,
        builder: (context, snap) {
          if (!snap.hasData) {
            return Center(child: snap.hasError ? Text(localizedError(snap.error!, l10n.t)) : const CircularProgressIndicator());
          }
          return _IdeasWall(
            ideas: snap.data!,
            capture: _capture,
            onCapture: _captureNow,
            onEdit: _edit,
            onDrop: _drop,
            onConvert: (idea) async {
              final ids = await context.read<Session>().api.convertIdea(idea.id);
              final prdId = ids['prdId'];
              if (prdId != null && context.mounted) {
                context.read<LoopFocus>().rememberPrd(prdId);
                context.go('/prd/$prdId');
              }
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
    return ListView(
      padding: const EdgeInsets.fromLTRB(28, 56, 28, 140),
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
        const SizedBox(height: 28),
        Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 720),
            child: Material(
              color: palette.surfaceRaised,
              shape: StadiumBorder(side: BorderSide(color: palette.strokeSubtle)),
              child: Padding(
                padding: const EdgeInsets.fromLTRB(20, 6, 6, 6),
                child: Row(
                  children: [
                    Expanded(
                      child: TextField(
                        controller: capture,
                        decoration: InputDecoration(
                          hintText: l10n.t('idea.captureHint'),
                          border: InputBorder.none,
                          enabledBorder: InputBorder.none,
                          focusedBorder: InputBorder.none,
                          filled: false,
                        ),
                        onSubmitted: (_) => onCapture(),
                      ),
                    ),
                    FilledButton(onPressed: onCapture, child: Text(l10n.t('idea.capture'))),
                  ],
                ),
              ),
            ),
          ),
        ),
        const SizedBox(height: 12),
        Center(
          child: Text(l10n.t('idea.privateHint'), style: theme.textTheme.bodySmall?.copyWith(color: palette.fgMuted), textAlign: TextAlign.center),
        ),
        const SizedBox(height: 28),
        if (ideas.isEmpty)
          Center(child: Text(emptyLabel, style: theme.textTheme.bodyLarge))
        else ...[
          Wrap(
            spacing: 16,
            runSpacing: 16,
            children: [
              for (final idea in ideas)
                SizedBox(
                  width: width >= 1100 ? (width - 88) / 3 : width >= 760 ? (width - 72) / 2 : width - 56,
                  child: _IdeaPoster(
                    idea: idea,
                    onEdit: () => onEdit(idea),
                    onDrop: () => onDrop(idea),
                    onConvert: () => onConvert(idea),
                    onOpenPrd: () => onOpenPrd(idea),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 20),
          Text(l10n.t('idea.noFolders'), style: theme.textTheme.labelSmall, textAlign: TextAlign.center),
        ],
      ],
    );
  }
}

class _IdeaPoster extends StatelessWidget {
  const _IdeaPoster({
    required this.idea,
    required this.onEdit,
    required this.onDrop,
    required this.onConvert,
    required this.onOpenPrd,
  });

  final Idea idea;
  final VoidCallback onEdit;
  final VoidCallback onDrop;
  final VoidCallback onConvert;
  final VoidCallback onOpenPrd;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final palette = Cobalt.of(context);
    return FieldCard(
      color: palette.surfaceRaised,
      padding: EdgeInsets.zero,
      child: Container(
        constraints: const BoxConstraints(minHeight: 240),
        decoration: BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [palette.surfaceRaised, palette.brandSubtle],
          ),
        ),
        child: Stack(
          children: [
            Positioned(
              right: -36,
              bottom: -48,
              child: Container(
                width: 160,
                height: 160,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: palette.brand.withValues(alpha: 0.08),
                ),
              ),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(22, 22, 12, 18),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(child: Text(idea.title.toUpperCase(), style: theme.textTheme.headlineSmall)),
                      PopupMenuButton<String>(
                        onSelected: (value) {
                          if (value == 'edit') onEdit();
                          if (value == 'drop') onDrop();
                        },
                        itemBuilder: (_) => [
                          PopupMenuItem(value: 'edit', child: Text(l10n.t('common.edit'))),
                          PopupMenuItem(value: 'drop', child: Text(l10n.t('idea.drop'))),
                        ],
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  Text(
                    idea.content.isEmpty ? idea.status : idea.content,
                    style: theme.textTheme.bodyLarge,
                    maxLines: 4,
                    overflow: TextOverflow.ellipsis,
                  ),
                  const SizedBox(height: 20),
                  ShadePill(label: l10n.t('idea.touched', {'when': timeAgo(idea.createdAt, l10n.t)})),
                  const SizedBox(height: 12),
                  TextButton(
                    onPressed: idea.prdId == null ? onConvert : onOpenPrd,
                    child: Text(idea.prdId == null ? l10n.t('idea.convert') : l10n.t('prd.title')),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class ShadePill extends StatelessWidget {
  const ShadePill({super.key, required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: palette.surfaceRaised,
        borderRadius: BorderRadius.circular(Cobalt.radiusPill),
        border: BorderSide(color: palette.strokeSubtle).toBorder(),
      ),
      child: Text(label, style: Theme.of(context).textTheme.labelSmall),
    );
  }
}

extension on BorderSide {
  BoxBorder toBorder() => Border.fromBorderSide(this);
}
