/// One HTTP call the Dio wrapper issues. Tests assert method/path/body.
class ApiCall {
  const ApiCall(this.method, this.path, [this.body]);

  final String method;
  final String path;
  final Map<String, dynamic>? body;
}

ApiCall patchWorkspace(
  String id, {
  required String title,
  required String description,
  String? coverUrl,
}) {
  return ApiCall('PATCH', '/workspaces/$id', {
    'title': title,
    'description': description,
    // Absent keeps the stored cover; the web has no cover control (PARA-09).
    'coverUrl': ?coverUrl,
  });
}

/// Deletes the signed-in account; the server asks for the current password.
ApiCall deleteMe(String password) =>
    ApiCall('DELETE', '/me', {'password': password});

ApiCall deleteWorkspace(String id) => ApiCall('DELETE', '/workspaces/$id');

ApiCall patchMemberRole(String workspaceId, String userId, String role) {
  return ApiCall('PATCH', '/workspaces/$workspaceId/members/$userId', {
    'role': role,
  });
}

ApiCall deleteMember(String workspaceId, String userId) {
  return ApiCall('DELETE', '/workspaces/$workspaceId/members/$userId');
}

ApiCall patchIdea(String id, {required String title, required String content}) {
  return ApiCall('PATCH', '/ideas/$id', {'title': title, 'content': content});
}

ApiCall deleteIdea(String id) => ApiCall('DELETE', '/ideas/$id');

ApiCall getIdea(String id) => ApiCall('GET', '/ideas/$id');

ApiCall getConversation(String id) => ApiCall('GET', '/conversations/$id');

ApiCall postPrdRevision(String id, {String? note}) {
  return ApiCall(
    'POST',
    '/prds/$id/revisions',
    note == null ? null : {'note': note},
  );
}

ApiCall postRoadmapNode(String projectId, String title, {String? parentId}) {
  return ApiCall('POST', '/projects/$projectId/roadmap', {
    'title': title,
    if (parentId != null) 'parentId': parentId,
  });
}

ApiCall patchProject(
  String id, {
  required String title,
  required String description,
  required String status,
  String? ownerId,
}) {
  return ApiCall('PATCH', '/projects/$id', {
    'title': title,
    'description': description,
    'status': status,
    // Absent keeps the owner; only an admin may change it (PARB-08).
    'ownerId': ?ownerId,
  });
}

ApiCall deleteProject(String id) => ApiCall('DELETE', '/projects/$id');

ApiCall patchProjectUpdate(String updateId, String content) {
  return ApiCall('PATCH', '/updates/$updateId', {'content': content});
}

ApiCall deleteProjectUpdate(String updateId) =>
    ApiCall('DELETE', '/updates/$updateId');

ApiCall patchRoadmapNode(
  String nodeId, {
  String? title,
  String? description,
  String? status,
  String? deadline,
  String? acceptance,
  String? ownerId,
  String? completionReason,
}) {
  return ApiCall('PATCH', '/roadmap/$nodeId', {
    'title': ?title,
    'description': ?description,
    'status': ?status,
    'deadline': ?deadline,
    'acceptance': ?acceptance,
    'ownerId': ?ownerId,
    'completionReason': ?completionReason,
  });
}

ApiCall moveRoadmapNode(
  String nodeId, {
  String? parentId,
  required int position,
}) {
  return ApiCall('POST', '/roadmap/$nodeId/move', {
    'parentId': parentId,
    'position': position,
  });
}

/// Deletes a node and its subtree. A subtree with evidence or done or
/// validated nodes answers 409 `roadmap_delete_has_history` with the counts;
/// [force] accepts that loss.
ApiCall deleteRoadmapNode(String nodeId, {bool force = false}) =>
    ApiCall('DELETE', '/roadmap/$nodeId${force ? '?force=true' : ''}');

/// Hands a node to a workspace member, on the record (an assignment row and
/// a feed update); the server answers the node view.
ApiCall delegateRoadmapNode(String nodeId, String toUserId, String note) =>
    ApiCall('POST', '/roadmap/$nodeId/delegate', {
      'toUserId': toUserId,
      'note': note,
    });

/// The current owner returns the node to whoever delegated it.
ApiCall handBackRoadmapNode(String nodeId, String note) =>
    ApiCall('POST', '/roadmap/$nodeId/handback', {'note': note});

/// Drafts sub-tasks for a node with the AI. Nothing is written: the response
/// is `{children: [{title, description?}]}`, to be sent on through
/// [confirmExpandRoadmapNode].
ApiCall expandRoadmapNode(String nodeId) =>
    ApiCall('POST', '/roadmap/$nodeId/expand');

/// Writes the [children] a person confirmed from the draft under the node
/// (`/expand` only drafts); the server answers 201 with the created nodes.
ApiCall confirmExpandRoadmapNode(
  String nodeId,
  List<Map<String, dynamic>> children,
) => ApiCall('POST', '/roadmap/$nodeId/expand/confirm', {'children': children});

ApiCall postPrdStatus(String id, String status) {
  return ApiCall('POST', '/prds/$id/status', {'status': status});
}

/// Starts the caller's own coach conversation on the PRD, or returns the one
/// they already have: 201 when new, 200 when resumed. Conversations are
/// owner-only, so `GET /prds/{id}/conversation` finds nothing for a teammate
/// who has not started one. [locale] is the app locale the coach must speak
/// ('en-US' / 'zh-CN'); the server canonicalizes and remembers it.
ApiCall postPrdConversation(String id, String locale) =>
    ApiCall('POST', '/prds/$id/conversation', {'locale': locale});

/// Confirms the stage the coach proposed (it waits in conversation meta,
/// not in the stage column): the server writes it and clears the proposal.
/// Declining is its own call, `declineConversationStage` in client_prd.dart
/// (`POST /conversations/{id}/stage/decline`): the server clears the proposal
/// and the finding resolutions parked with it, and the row keeps the old stage.
ApiCall postConversationStage(String id, String stage) =>
    ApiCall('POST', '/conversations/$id/stage', {'stage': stage});

/// Drafts a branched roadmap from the PRD. Nothing is written: the response
/// is `{tree}` (plus `projectId` only when the PRD is already linked), to be
/// sent on through [postPrdBuildConfirm]. An unapproved PRD answers 409
/// `prd_not_approved`.
ApiCall postPrdBuild(String id) => ApiCall('POST', '/prds/$id/build');

/// Writes a drafted [tree] (`/build` only drafts). When the project already
/// has roadmap nodes the server answers 409 `replace_conflict` with
/// `liveCount` (and `evidenceCount`); send that back as [expectedCount] to
/// replace them. A roadmap with evidence or done/validated nodes answers 409
/// `roadmap_replace_has_history` with the counts; [force] accepts that loss.
ApiCall postPrdBuildConfirm(
  String id,
  Map<String, dynamic> tree, {
  int? expectedCount,
  bool force = false,
}) {
  return ApiCall('POST', '/prds/$id/build/confirm', {
    'tree': tree,
    'expectedCount': ?expectedCount,
    if (force) 'force': true,
  });
}

/// `?root=` when the read covers one subtree only (rv5-PARC-12).
String _root(String? rootId) =>
    rootId == null ? '' : '?root=${Uri.encodeQueryComponent(rootId)}';

ApiCall getRoadmap(String projectId, {String? rootId}) => ApiCall(
  'GET',
  '/projects/$projectId/roadmap'
      '${_root(rootId)}',
);

ApiCall getRoadmapNode(String nodeId) => ApiCall('GET', '/roadmap/$nodeId');

ApiCall getNodeAssignments(String nodeId) =>
    ApiCall('GET', '/roadmap/$nodeId/assignments');

ApiCall getNodeEvidence(String nodeId) =>
    ApiCall('GET', '/roadmap/$nodeId/evidence');

ApiCall getRoadmapProgress(String projectId, {String? rootId}) => ApiCall(
  'GET',
  '/projects/$projectId/roadmap/progress'
      '${_root(rootId)}',
);

/// One human-entered validation observation; [observedAt] is sent as its date.
ApiCall postNodeEvidence(
  String nodeId, {
  required String detail,
  required String source,
  required DateTime observedAt,
}) => ApiCall('POST', '/roadmap/$nodeId/evidence', {
  'detail': detail,
  'sourceUrl': source,
  'observedAt': observedAt.toIso8601String().substring(0, 10),
});

ApiCall postValidate(String nodeId, String reason) =>
    ApiCall('POST', '/roadmap/$nodeId/validate', {'reason': reason});

ApiCall postCursor(
  String projectId,
  double x,
  double y, {
  required String clientId,
}) => ApiCall(
  'POST',
  '/projects/$projectId/cursor?client=${Uri.encodeQueryComponent(clientId)}',
  {'x': x, 'y': y},
);

ApiCall putProjectRepo(String projectId, String repoUrl) =>
    ApiCall('PUT', '/projects/$projectId/repo', {'repoUrl': repoUrl});

/// The page of updates older than [before] (a nextCursor), newest first.
ApiCall getProjectUpdates(String projectId, {String? before}) => ApiCall(
  'GET',
  '/projects/$projectId/updates'
      '${before == null ? '' : '?before=${Uri.encodeQueryComponent(before)}'}',
);

/// The scaffolded insideout.yaml matching guide, as text/yaml.
ApiCall getProjectGuide(String projectId) =>
    ApiCall('GET', '/projects/$projectId/guide');

ApiCall postSyncGithub(String projectId) =>
    ApiCall('POST', '/projects/$projectId/sync-github');

ApiCall postProjectUpdate(String projectId, String kind, String content) =>
    ApiCall('POST', '/projects/$projectId/updates', {
      'kind': kind,
      'content': content,
    });

ApiCall postProposalDecision(
  String updateId, {
  required bool accept,
  required String reason,
  required bool apply,
}) => ApiCall('POST', '/agent/proposals/$updateId/decision', {
  'decision': accept ? 'accepted' : 'rejected',
  'reason': reason,
  'apply': apply,
});
