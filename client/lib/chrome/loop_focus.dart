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

  void rememberWorkspace(String id) {
    workspaceId = id;
  }

  void rememberProject(String id, {String? workspaceId}) {
    projectId = id;
    if (workspaceId != null) this.workspaceId = workspaceId;
  }

  void rememberPrd(String id) {
    prdId = id;
  }

  void clear() {
    workspaceId = null;
    projectId = null;
    prdId = null;
    notifyListeners();
  }
}
