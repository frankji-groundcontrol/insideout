import 'dart:async';

import 'package:flutter/foundation.dart';

import '../../api/client.dart';
import '../../api/errors.dart';
import '../../api/models.dart';
import 'roadmap_canvas.dart';
import 'roadmap_tree.dart';

/// The roadmap page's data between its State and the API client (STRC-05):
/// the nodes, the progress reading, the opened node's read, the project
/// view, and the live half — the progress and presence streams with their
/// retry back-off, and the shared cursors. One controller serves the whole
/// tree ([rootId] null), an opened subtree, and the Mine view over either.
/// The page keeps only layout and local UI state; it listens and rebuilds.
class RoadmapController extends ChangeNotifier {
  RoadmapController(
    this._api, {
    required this.projectId,
    this.rootId,
    this.live = true,
  }) {
    project = _api.getProject(projectId);
    _fetch();
    project
        .then((p) {
          if (_stop.isCompleted) return;
          planWriter = p.project.canWritePlan;
          notifyListeners();
        })
        // A failed read leaves the plan-writer items hidden; the server
        // refuses them anyway.
        .catchError((_) {});
    // A node's window holds no stream: a browser gives one server only six
    // connections, and every open stream keeps one (BUG 2026-11-22).
    if (!live) return;
    // The live progress reading is project-wide: a subtree keeps its own.
    if (rootId == null) _progressLoop();
    _presenceLoop();
  }

  /// Whether this controller holds the live streams; a node's window does
  /// not, and reloads on the page's [changes] instead.
  final bool live;

  /// Ticks on every live progress reading: the roadmap changed somewhere.
  final changes = ValueNotifier<int>(0);

  final ApiClient _api;
  final String projectId;

  /// The node opened as its own tree; null for the whole project.
  final String? rootId;

  late Future<List<RoadmapNode>> nodes;
  late Future<RoadmapProgress> progress;

  /// The opened node's read (ancestors and project); null for the whole tree.
  Future<RoadmapNodeRead>? read;
  late final Future<ProjectDetail> project;

  /// The caller writes the whole plan (GET /projects/{pid} canWritePlan).
  bool planWriter = false;

  /// The opened node's canWrite: "New item" adds under it.
  bool rootWritable = false;

  /// The opened node's read once it landed: its ancestors carry the owners
  /// its tree inherits, for Mine and the inherited owner label.
  RoadmapNodeRead? opened;

  /// The opened node's person in charge and their first glance at it, once
  /// it landed; null for the whole tree, before then, or when it failed.
  RoadmapGlance? glance;

  /// The whole project's nodes, read once when someone else's view needs
  /// their nodes outside the opened tree; the project route reuses [nodes].
  Future<List<RoadmapNode>>? _projectNodes;
  Future<List<RoadmapNode>> get projectNodes =>
      _projectNodes ??= rootId == null ? nodes : _api.listRoadmap(projectId);

  /// Member names by user id, read only when an ancestor above the opened
  /// tree has an owner (the ancestor refs carry the id alone).
  Map<String, String> names = const {};

  /// Titles of nodes outside the loaded list that a loaded node waits on,
  /// read once from the whole project so the Canvas can name them.
  Map<String, String> outsideTitles = const {};

  /// The newest reading the progress stream pushed; null until it does and
  /// after a reload, so the fetched reading shows.
  RoadmapProgress? liveProgress;
  List<PresenceEntry> presence = [];
  final Map<String, CursorDot> cursors = {};

  DateTime _lastCursorSent = DateTime.fromMillisecondsSinceEpoch(0);
  final _retryTimers = <Timer>{};

  /// Completed on dispose: the live streams close their connections with it.
  final _stop = Completer<void>();

  /// Completed while the tab is hidden: the streams close until it shows.
  Completer<void> _hidden = Completer<void>();
  Completer<void> _shown = Completer<void>()..complete();

  /// A hidden tab lets its streams go, so other tabs keep connections; on
  /// showing again it reconnects and reloads what it missed.
  void setVisible(bool visible) {
    if (visible == _shown.isCompleted || !_live) return;
    if (visible) {
      _hidden = Completer<void>();
      _shown.complete();
      reload();
    } else {
      _shown = Completer<void>();
      _hidden.complete();
    }
  }

  Future<void> get _sessionStop => Future.any([_stop.future, _hidden.future]);
  late final String _clientId = DateTime.now().microsecondsSinceEpoch
      .toRadixString(36);

  bool get _live => !_stop.isCompleted;

  void _changed() {
    if (_live) notifyListeners();
  }

  void _fetch() {
    _projectNodes = null;
    final root = rootId;
    nodes = root == null
        ? _api.listRoadmap(projectId)
        : _api.listRoadmapSubtree(projectId, root);
    progress = root == null
        ? _api.roadmapProgress(projectId)
        : _api.roadmapSubtreeProgress(projectId, root);
    read = root == null ? null : _api.readRoadmapNode(root);
    nodes.then(_nameOutside, onError: (_) {});
    if (root != null) {
      _api
          .roadmapGlance(root)
          .then((g) {
            glance = g;
            _changed();
          })
          // Without it the page shows no banner and keeps Mine.
          .catchError((_) {});
    }
    read
        ?.then((r) {
          if (!_live) return;
          rootWritable = r.node.canWrite;
          opened = r;
          _changed();
          if (names.isEmpty && r.ancestors.any((a) => a.ownerId != null)) {
            _loadNames();
          }
        })
        .catchError((_) {});
  }

  void _nameOutside(List<RoadmapNode> loaded) {
    final ids = {for (final n in loaded) n.id};
    bool unnamed(String id) =>
        !ids.contains(id) && !outsideTitles.containsKey(id);
    if (!loaded.any((n) => n.dependsOn.any(unnamed))) return;
    _api
        .listRoadmap(projectId)
        .then((all) {
          outsideTitles = {for (final n in all) n.id: n.title};
          _changed();
        })
        // Unnamed, the line reads "another item".
        .catchError((_) {});
  }

  void _loadNames() => project
      .then((p) => _api.listMembers(p.project.workspaceId))
      .then((members) {
        names = {for (final m in members) m.userId: m.username};
        _changed();
      })
      // Unnamed, the label reads "someone".
      .catchError((_) {});

  /// Reads the nodes, the progress and the opened node again.
  Future<void> reload() async {
    _fetch();
    liveProgress = null;
    _changed();
    await Future.wait([nodes, progress, ?read]);
  }

  /// Mine over [all]: the caller's branches, or the whole opened tree when
  /// an ancestor above it is theirs.
  List<RoadmapNode> mine(List<RoadmapNode> all, String userId) =>
      opened?.ancestors.any((a) => a.ownerId == userId) == true
      ? all
      : roadmapMine(all, userId);

  /// Shares the pointer at most every 200ms.
  void sendCursor(double x, double y) {
    final now = DateTime.now();
    if (now.difference(_lastCursorSent).inMilliseconds < 200) return;
    _lastCursorSent = now;
    _api.cursor(projectId, x, y, clientId: _clientId).catchError((_) {});
  }

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

  /// Runs [session] again each time it ends, until dispose: wait 500ms,
  /// doubling to 5s, between runs; 401/403/404 are final.
  Future<void> _retrying(Future<void> Function() session) async {
    var backoff = const Duration(milliseconds: 500);
    while (_live) {
      if (!_shown.isCompleted) {
        await Future.any([_shown.future, _stop.future]);
        if (!_live) return;
        backoff = const Duration(milliseconds: 500);
      }
      try {
        await session();
      } on ApiException catch (e) {
        if (e.status == 401 || e.status == 403 || e.status == 404) return;
      } catch (_) {}
      if (!_live) return;
      if (!_shown.isCompleted) continue;
      await _backoff(backoff);
      backoff = backoff * 2 > const Duration(seconds: 5)
          ? const Duration(seconds: 5)
          : backoff * 2;
    }
  }

  Future<void> _progressLoop() => _retrying(
    () => _api.progressStream(projectId, (p) {
      liveProgress = p;
      changes.value++;
      _changed();
    }, stop: _sessionStop),
  );

  Future<void> _presenceLoop() => _retrying(
    () => _api.presenceStream(
      projectId,
      _clientId,
      (entries) {
        final live = entries.map((e) => e.sessionId).toSet();
        cursors.removeWhere((id, _) => !live.contains(id));
        presence = entries;
        _changed();
      },
      onCursor: (sessionId, name, x, y) {
        cursors[sessionId] = CursorDot(
          sessionId: sessionId,
          name: name,
          x: x,
          y: y,
        );
        _changed();
      },
      stop: _sessionStop,
    ),
  );

  @override
  void dispose() {
    _stop.complete();
    for (final t in _retryTimers) {
      t.cancel();
    }
    changes.dispose();
    super.dispose();
  }
}
