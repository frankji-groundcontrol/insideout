import 'package:flutter/material.dart';

import '../../theme/block_chrome.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_theme.dart';

/// Pull-to-refresh without the round spinner. [RefreshIndicator.noSpinner]
/// keeps what the gesture already does — the arm-and-release threshold, the
/// timing, the call to [onRefresh] — and paints nothing; the indicator is the
/// square [BlockLoader] on a raised block instead (DESIGN.md: loaders are
/// square blocks, never circles).
///
/// The block shows from the moment the pull arms until the refresh settles,
/// so the pull gets "release now" feedback and the wait gets a busy state.
class BlockRefresh extends StatefulWidget {
  const BlockRefresh({
    super.key,
    required this.label,
    required this.onRefresh,
    required this.child,
  });

  /// Localized screen-reader name; the stock indicator announced one too.
  final String label;
  final Future<void> Function() onRefresh;
  final Widget child;

  @override
  State<BlockRefresh> createState() => _BlockRefreshState();
}

class _BlockRefreshState extends State<BlockRefresh> {
  RefreshIndicatorStatus? _status;

  // Flutter reports drag, armed, snap, done and canceled; the refresh phase
  // follows snap without a callback of its own, so snap holds until done.
  bool get _showing =>
      _status == RefreshIndicatorStatus.armed ||
      _status == RefreshIndicatorStatus.snap ||
      _status == RefreshIndicatorStatus.refresh;

  @override
  Widget build(BuildContext context) {
    final palette = Cobalt.of(context);
    return Stack(
      fit: StackFit.expand,
      children: [
        RefreshIndicator.noSpinner(
          onRefresh: widget.onRefresh,
          onStatusChange: (status) => setState(() => _status = status),
          child: widget.child,
        ),
        if (_showing)
          Align(
            alignment: Alignment.topCenter,
            child: Padding(
              padding: const EdgeInsets.only(top: Cobalt.space3),
              child: Semantics(
                label: widget.label,
                liveRegion: true,
                child: Container(
                  padding: const EdgeInsets.all(Cobalt.space3),
                  decoration: BoxDecoration(
                    color: palette.surfaceRaised,
                    border: Border.all(
                      color: palette.strokeSubtle,
                      width: Cobalt.borderContainer,
                    ),
                    boxShadow: cast(Theme.of(context).brightness),
                  ),
                  child: const BlockLoader(size: Cobalt.space6),
                ),
              ),
            ),
          ),
      ],
    );
  }
}
