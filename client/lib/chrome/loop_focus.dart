import 'package:flutter/foundation.dart';

/// Last product-loop coordinates so the bottom dock can move between
/// Ideas, Version, Commit, Roadmap, and Evidence without new routes.
///
/// Setters are silent: pages seed IDs from `initState` during the first
/// build, and the dock reads the values on tap.
class LoopFocus extends ChangeNotifier {
  String? workspaceId;
  String? projectId;
  String? prdId;

  /// Whether the project's working version has a human commit. Null until a
  /// page that loaded the project says so.
  bool? prdCommitted;

  void rememberWorkspace(String id) {
    workspaceId = id;
  }

  void rememberProject(String id, {String? workspaceId, bool? prdCommitted}) {
    projectId = id;
    if (workspaceId != null) this.workspaceId = workspaceId;
    if (prdCommitted != null && this.prdCommitted != prdCommitted) {
      this.prdCommitted = prdCommitted;
      notifyListeners();
    }
  }

  void rememberPrd(String id, {bool? committed}) {
    final same = prdId == id;
    prdId = id;
    // A different PRD has not been checked yet, so the previous answer
    // must not keep marking Commit.
    if (!same && committed == null && prdCommitted != null) {
      prdCommitted = null;
      notifyListeners();
    } else if (committed != null && prdCommitted != committed) {
      prdCommitted = committed;
      notifyListeners();
    }
  }

  void clear() {
    workspaceId = null;
    projectId = null;
    prdId = null;
    prdCommitted = null;
    notifyListeners();
  }
}
