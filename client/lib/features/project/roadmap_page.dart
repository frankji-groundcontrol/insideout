import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../app.dart';
import '../../chrome/loop_focus.dart';
import '../../session/appearance.dart';
import '../../session/session.dart';
import '../../theme/block_button.dart';
import '../../theme/block_form.dart';
import '../../theme/block_state.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_chip.dart';
import '../../widgets/guarded_action.dart';
import 'roadmap_bands.dart';
import 'roadmap_breadcrumb.dart';
import 'roadmap_canvas.dart';
import 'roadmap_controller.dart';
import 'roadmap_header.dart';
import 'roadmap_lens_banner.dart';
import 'roadmap_node_card.dart';
import 'roadmap_progress_view.dart';
import 'roadmap_step_flow.dart';
import 'roadmap_tree.dart';
import 'roadmap_window_layer.dart';

part 'roadmap_page_views.dart';

class RoadmapPage extends StatefulWidget {
  const RoadmapPage({
    super.key,
    required this.projectId,
    this.rootId,
    this.view,
    this.embedded = false,
    this.onReroot,
  });

  final String projectId;

  /// The node opened as its own tree (/projects/:id/roadmap/:nodeId): the
  /// page shows it and the nodes below it. Null for the whole project.
  final String? rootId;

  /// The view to open on (`?view=canvas`). Progress when absent or
  /// unknown, as an opened node opens.
  final String? view;

  /// Inside a node's window or phone sheet: no app chrome or dock, the view
  /// toggles in the header, and nodes open on the page's window layer.
  final bool embedded;

  /// A breadcrumb ancestor inside a window: the window re-roots there.
  final void Function(String nodeId)? onReroot;

  @override
  State<RoadmapPage> createState() => _RoadmapPageState();
}

/// Mine is the canvas over the caller's own branches (owner decision 2: a
/// default view, not a wall); on a node someone else is in charge of it is
/// theirs ("Priya's").
enum _RoadmapView { progress, bands, canvas, mine }

class _RoadmapPageState extends State<RoadmapPage> {
  /// The nodes, progress, presence and cursors; this State keeps the layout
  /// and local UI state only.
  late final RoadmapController _c;
  late _RoadmapView _view = _RoadmapView.values.firstWhere(
    (v) => v.name == widget.view,
    orElse: () => _RoadmapView.progress,
  );

  @override
  void initState() {
    super.initState();
    context.read<LoopFocus>().rememberProject(widget.projectId);
    _c = RoadmapController(
      context.read<Session>().api,
      projectId: widget.projectId,
      rootId: widget.rootId,
    )..addListener(_changed);
  }

  void _changed() => setState(() {});

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  /// "New item": a top-level goal for a plan writer, or in an opened node a
  /// child of that node for whoever may write it.
  Future<void> _addRoot() async {
    final t = context.read<Appearance>().t;
    final values = await showBlockForm(
      context: context,
      title: t(widget.rootId == null ? 'roadmap.addRoot' : 'roadmap.addChild'),
      confirmLabel: t('roadmap.add'),
      fields: [
        BlockFormField(label: t('roadmap.rootPlaceholder'), required: true),
      ],
    );
    if (values == null || !mounted) return;
    final title = values.first.trim();
    if (title.isEmpty) return;
    final done = await guardWrite(
      context,
      () => context.read<Session>().api.createRoadmapNode(
        widget.projectId,
        title,
        parentId: widget.rootId,
      ),
    );
    if (done && mounted) await _c.reload();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final choices = _choices(l10n);
    // Four toggles plus four utility icons run off a phone's left edge, so
    // there the toggles move into the header; a window has no utility block.
    final narrow = MediaQuery.sizeOf(context).width < Cobalt.bpNarrow;
    final header = _header(choices, narrow || widget.embedded);
    final body = _view == _RoadmapView.progress
        ? _progressBody(l10n)
        : _treeBody(l10n);
    // Progress and Bands scroll the header away with their content (roadmap
    // critique 2026-11-21); the canvas pans under a pinned one, which
    // scrolls itself past most of a short window.
    final page = _view == _RoadmapView.progress || _view == _RoadmapView.bands
        ? NestedScrollView(
            headerSliverBuilder: (context, _) => [
              SliverToBoxAdapter(child: header),
            ],
            body: body,
          )
        : LayoutBuilder(
            builder: (context, box) => Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                ConstrainedBox(
                  constraints: BoxConstraints(maxHeight: box.maxHeight * 0.6),
                  child: SingleChildScrollView(child: header),
                ),
                Expanded(child: body),
              ],
            ),
          );
    if (widget.embedded) return page;
    return AppScaffold(
      actions: narrow ? null : choices,
      body: RoadmapWindowLayer(
        projectId: widget.projectId,
        resolve: _resolve,
        child: page,
      ),
    );
  }

  /// Progress, Bands, Canvas, and Mine or the person in charge's.
  List<Widget> _choices(Appearance l10n) {
    final them = _theirs;
    return [
      for (final view in _RoadmapView.values)
        BlockChoice(
          label: l10n.t(
            switch (view) {
              _RoadmapView.progress => 'roadmap.viewProgress',
              _RoadmapView.bands => 'roadmap.viewBands',
              _RoadmapView.canvas => 'roadmap.viewCanvas',
              _RoadmapView.mine =>
                them == null ? 'roadmap.viewMine' : 'roadmap.viewTheirs',
            },
            {'name': them?.firstName},
          ),
          selected: _view == view,
          onTap: () => setState(() => _view = view),
        ),
    ];
  }

  /// The title block, then who else is here: one chip per user, however
  /// many sessions they hold open.
  Widget _header(List<Widget> choices, bool narrow) {
    final people = {for (final e in _c.presence) e.userId: e.name};
    final glance = _c.glance;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        RoadmapHeader(
          compact: narrow,
          top: widget.embedded ? Cobalt.space3 : 64,
          windowed: widget.embedded,
          viewChoices: choices,
          onAdd: (widget.rootId == null ? _c.planWriter : _c.rootWritable)
              ? _addRoot
              : null,
          breadcrumb: _c.read == null
              ? null
              : RoadmapBreadcrumb(
                  projectId: widget.projectId,
                  read: _c.read!,
                  onAncestor: widget.onReroot,
                ),
          title: widget.rootId == null ? null : _c.opened?.node.title,
        ),
        if (people.isNotEmpty)
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 6),
            child: Wrap(
              spacing: Cobalt.space2,
              children: [
                Icon(
                  Icons.groups_2_outlined,
                  size: 16,
                  color: Cobalt.of(context).brand,
                ),
                for (final name in people.values)
                  ShadeChip(label: name, tone: ShadeTone.neutral),
              ],
            ),
          ),
        if (glance != null)
          // Below the window layer, so a Now item opens in a window.
          Builder(
            builder: (context) => RoadmapLensBanner(
              glance: glance,
              full: widget.embedded,
              viewerId: context.read<Session>().user?.id ?? '',
              onOpen: (item) =>
                  RoadmapWindowScope.maybeOf(context)?.open(item.id),
            ),
          ),
      ],
    );
  }
}
