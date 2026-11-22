import 'dart:math' as math;

import 'package:flutter/foundation.dart';
import 'package:flutter/painting.dart';

import '../../api/models.dart';

/// One node's floating window: the node it shows, the owner it inherits, and
/// where it sits over the roadmap page.
class RoadmapWindow {
  RoadmapWindow({required this.node, this.inheritedOwner, required this.rect});

  final RoadmapNode node;
  final ({String name, String via})? inheritedOwner;
  Rect rect;
}

/// The node windows open over the roadmap page, back to front: the last one
/// is in front. Pure layout state; every move takes the viewport it lives in.
class RoadmapWindows extends ChangeNotifier {
  static const minSize = Size(320, 240);
  static const defaultSize = Size(440, 520);

  /// The step each new window steps down and right from the last.
  static const cascade = 32.0;

  /// The title bar's height: a dragged window keeps it in view.
  static const titleBar = 48.0;

  /// Where the first window sits from the top right: below the page's
  /// utility block and clear of the Canvas's zoom block.
  static const _top = 96.0;
  static const _rightGap = 72.0;

  final windows = <RoadmapWindow>[];

  RoadmapWindow? _of(String id) =>
      windows.where((w) => w.node.id == id).firstOrNull;

  /// Opens [node]'s window cascaded from the front one, or brings its open
  /// window to the front.
  void open(
    RoadmapNode node,
    Size view, {
    ({String name, String via})? inheritedOwner,
  }) {
    if (_of(node.id) != null) return focus(node.id);
    final size = Size(
      math.min(defaultSize.width, view.width),
      math.min(defaultSize.height, view.height),
    );
    final first = Offset(view.width - size.width - _rightGap, _top);
    final last = windows.lastOrNull?.rect.topLeft;
    var at = last == null ? first : last + const Offset(cascade, cascade);
    // Past the viewport's edge the cascade starts over.
    if (at.dx + size.width > view.width || at.dy + size.height > view.height) {
      at = first;
    }
    windows.add(
      RoadmapWindow(
        node: node,
        inheritedOwner: inheritedOwner,
        rect: _clamp(at & size, view, whole: true),
      ),
    );
    notifyListeners();
  }

  /// Brings [id]'s window to the front.
  void focus(String id) {
    final w = _of(id);
    if (w == null || identical(w, windows.last)) return;
    windows
      ..remove(w)
      ..add(w);
    notifyListeners();
  }

  void close(String id) {
    windows.removeWhere((w) => w.node.id == id);
    notifyListeners();
  }

  /// Closes the front window; false when none is open.
  bool closeFront() {
    if (windows.isEmpty) return false;
    windows.removeLast();
    notifyListeners();
    return true;
  }

  /// Drags [id]'s window by [delta], its title bar kept in [view].
  void move(String id, Offset delta, Size view) {
    final w = _of(id);
    if (w == null) return;
    w.rect = _clamp(w.rect.shift(delta), view);
    notifyListeners();
  }

  /// Grows [id]'s window by [delta] from its bottom-right, between the
  /// minimum size and the viewport's edge.
  void resize(String id, Offset delta, Size view) {
    final w = _of(id);
    if (w == null) return;
    final r = w.rect;
    double fit(double v, double min, double room) =>
        v.clamp(min, math.max(min, room));
    w.rect = Rect.fromLTWH(
      r.left,
      r.top,
      fit(r.width + delta.dx, minSize.width, view.width - r.left),
      fit(r.height + delta.dy, minSize.height, view.height - r.top),
    );
    notifyListeners();
  }

  /// [r] moved so it lies across the viewport and its title bar, or when
  /// [whole] all of it, shows.
  static Rect _clamp(Rect r, Size view, {bool whole = false}) => Rect.fromLTWH(
    r.left.clamp(0, math.max(0, view.width - r.width)),
    r.top.clamp(0, math.max(0, view.height - (whole ? r.height : titleBar))),
    r.width,
    r.height,
  );
}
