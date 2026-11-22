import 'dart:async';

import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';

import '../../api/models.dart';
import '../../theme/cobalt.dart';
import 'roadmap_page.dart';
import 'roadmap_window_frame.dart';
import 'roadmap_windows.dart';

/// How a node on the roadmap page, or in a window over it, opens its window:
/// [tap] opens it after a double-click's wait and drills into the node on a
/// second tap; [open] opens it now; [drill] goes to the node's own roadmap.
class RoadmapWindowScope extends InheritedWidget {
  const RoadmapWindowScope({
    super.key,
    required this.tap,
    required this.open,
    required this.drill,
    required super.child,
  });

  final void Function(String nodeId) tap;
  final void Function(String nodeId) open;
  final void Function(String nodeId) drill;

  /// Read at tap time, so the nodes do not rebuild with the layer.
  static RoadmapWindowScope? maybeOf(BuildContext context) =>
      context.getInheritedWidgetOfExactType<RoadmapWindowScope>();

  @override
  bool updateShouldNotify(RoadmapWindowScope old) => false;
}

/// The roadmap page's window layer: [child] with the node windows floating
/// over it, each the node's own roadmap page embedded. Below the narrow
/// breakpoint a node opens as a full-screen sheet instead. Windows live with
/// the page, so another roadmap route closes them.
class RoadmapWindowLayer extends StatefulWidget {
  const RoadmapWindowLayer({
    super.key,
    required this.projectId,
    required this.resolve,
    required this.child,
  });

  final String projectId;

  /// The loaded node for an id; null when the page has not loaded it.
  final Future<RoadmapNode?> Function(String nodeId) resolve;
  final Widget child;

  @override
  State<RoadmapWindowLayer> createState() => _RoadmapWindowLayerState();
}

class _RoadmapWindowLayerState extends State<RoadmapWindowLayer> {
  final _windows = RoadmapWindows();
  var _view = Size.zero;
  Timer? _pending;
  String? _pendingId;

  @override
  void initState() {
    super.initState();
    _windows.addListener(_changed);
    HardwareKeyboard.instance.addHandler(_key);
  }

  void _changed() => setState(() {});

  @override
  void dispose() {
    HardwareKeyboard.instance.removeHandler(_key);
    _pending?.cancel();
    _windows.dispose();
    super.dispose();
  }

  /// Escape closes the front window while the page is the top route, so an
  /// open menu or dialog takes it first.
  bool _key(KeyEvent e) {
    if (e is! KeyDownEvent || e.logicalKey != LogicalKeyboardKey.escape) {
      return false;
    }
    if (!(ModalRoute.of(context)?.isCurrent ?? true)) return false;
    return _windows.closeFront();
  }

  /// A first tap waits out a double-click; a second tap on the same node
  /// in that wait drills in instead.
  void _tap(String id) {
    final prior = _pendingId;
    _pending?.cancel();
    _pendingId = null;
    if (prior == id) return _drill(id);
    if (prior != null) _open(prior);
    _pendingId = id;
    _pending = Timer(kDoubleTapTimeout, () {
      _pendingId = null;
      _open(id);
    });
  }

  Future<void> _open(String id) async {
    final found = await widget.resolve(id);
    if (found == null || !mounted) return;
    if (MediaQuery.sizeOf(context).width < Cobalt.bpNarrow) {
      return _sheet(found);
    }
    _windows.open(found, _view);
  }

  /// A breadcrumb inside [id]'s window roots it at the ancestor [to].
  Future<void> _reroot(String id, String to) async {
    final found = await widget.resolve(to);
    if (found != null && mounted) _windows.reroot(id, found);
  }

  void _drill(String id) => GoRouter.maybeOf(
    context,
  )?.go('/projects/${widget.projectId}/roadmap/$id');

  /// The narrow screen's window: the same chrome and page, full screen,
  /// with no drag or resize; a breadcrumb re-roots it in place.
  Future<void> _sheet(RoadmapNode first) {
    var node = first;
    return showDialog<void>(
      context: context,
      builder: (_) => _scope(
        StatefulBuilder(
          builder: (ctx, setSheet) => Dialog.fullscreen(
            child: RoadmapWindowFrame(
              node: node,
              onClose: () => Navigator.of(ctx).pop(),
              onOpenRoadmap: () {
                Navigator.of(ctx).pop();
                _drill(node.id);
              },
              body: _page(node, (to) async {
                final found = await widget.resolve(to);
                if (found != null && ctx.mounted) setSheet(() => node = found);
              }),
            ),
          ),
        ),
      ),
    );
  }

  /// [node]'s own roadmap page, embedded; its breadcrumb calls [reroot].
  Widget _page(RoadmapNode node, void Function(String to) reroot) =>
      RoadmapPage(
        key: ValueKey('roadmap-window-page-${node.id}'),
        projectId: widget.projectId,
        rootId: node.id,
        embedded: true,
        onReroot: reroot,
      );

  Widget _window(RoadmapWindow w) {
    final id = w.node.id;
    return Positioned.fromRect(
      key: ValueKey('roadmap-window-slot-$id'),
      rect: w.rect,
      child: RoadmapWindowFrame(
        node: w.node,
        onFocus: () => _windows.focus(id),
        onClose: () => _windows.close(id),
        onOpenRoadmap: () => _drill(id),
        onMove: (d) => _windows.move(id, d, _view),
        onResize: (d) => _windows.resize(id, d, _view),
        body: _page(w.node, (to) => _reroot(id, to)),
      ),
    );
  }

  /// The dialog route of a sheet sits outside this layer, so its page gets
  /// the scope again.
  Widget _scope(Widget child) =>
      RoadmapWindowScope(tap: _tap, open: _open, drill: _drill, child: child);

  @override
  Widget build(BuildContext context) => _scope(
    LayoutBuilder(
      builder: (context, box) {
        _view = box.biggest;
        return Stack(
          children: [
            Positioned.fill(child: widget.child),
            for (final w in _windows.windows) _window(w),
          ],
        );
      },
    ),
  );
}
