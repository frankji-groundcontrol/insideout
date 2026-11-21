part of 'roadmap_page.dart';

/// The roadmap page's live half: the progress and presence streams with
/// their retry back-off, and the shared cursors.
mixin _RoadmapLive on State<RoadmapPage> {
  RoadmapProgress? _liveProgress;
  List<PresenceEntry> _presence = [];
  final Map<String, CursorDot> _cursors = {};
  DateTime _lastCursorSent = DateTime.fromMillisecondsSinceEpoch(0);
  final _retryTimers = <Timer>{};

  /// Completed on dispose: the live streams close their connections with it.
  final _stop = Completer<void>();
  late final String _clientId = DateTime.now().microsecondsSinceEpoch
      .toRadixString(36);

  Future<void> _backoff(Duration d) {
    final done = Completer<void>();
    late final Timer t;
    t = Timer(d, () {
      _retryTimers.remove(t);
      done.complete();
    });
    _retryTimers.add(t);
    return done.future;
  }

  void _sendCursor(double x, double y) {
    final now = DateTime.now();
    if (now.difference(_lastCursorSent).inMilliseconds < 200) return;
    _lastCursorSent = now;
    context
        .read<Session>()
        .api
        .cursor(widget.projectId, x, y, clientId: _clientId)
        .catchError((_) {});
  }

  /// Runs [session] again each time it ends, until the page goes away: wait
  /// 500ms, doubling to 5s, between runs; 401/403/404 are final.
  Future<void> _retrying(Future<void> Function() session) async {
    var backoff = const Duration(milliseconds: 500);
    while (mounted) {
      try {
        await session();
      } on ApiException catch (e) {
        if (e.status == 401 || e.status == 403 || e.status == 404) return;
      } catch (_) {}
      if (!mounted) return;
      await _backoff(backoff);
      backoff = backoff * 2 > const Duration(seconds: 5)
          ? const Duration(seconds: 5)
          : backoff * 2;
    }
  }

  Future<void> _progressLoop(ApiClient api) => _retrying(
    () => api.progressStream(widget.projectId, (progress) {
      if (!mounted) return;
      setState(() => _liveProgress = progress);
    }, stop: _stop.future),
  );

  Future<void> _presenceLoop(ApiClient api) => _retrying(
    () => api.presenceStream(
      widget.projectId,
      _clientId,
      (entries) {
        if (!mounted) return;
        final live = entries.map((e) => e.sessionId).toSet();
        _cursors.removeWhere((id, _) => !live.contains(id));
        setState(() => _presence = entries);
      },
      onCursor: (sessionId, name, x, y) {
        if (!mounted) return;
        setState(
          () => _cursors[sessionId] = CursorDot(
            sessionId: sessionId,
            name: name,
            x: x,
            y: y,
          ),
        );
      },
      stop: _stop.future,
    ),
  );

  @override
  void dispose() {
    _stop.complete();
    for (final t in _retryTimers) {
      t.cancel();
    }
    super.dispose();
  }
}
