/// One HTTP call the Dio wrapper issues. Tests assert method/path/body.
class ApiCall {
  const ApiCall(this.method, this.path, [this.body]);

  final String method;
  final String path;
  final Map<String, dynamic>? body;
}

ApiCall patchWorkspace(String id, {required String title, required String description, String? coverUrl}) {
  return ApiCall('PATCH', '/workspaces/$id', {
    'title': title,
    'description': description,
    'coverUrl': coverUrl,
  });
}

ApiCall deleteWorkspace(String id) => ApiCall('DELETE', '/workspaces/$id');

ApiCall patchMemberRole(String workspaceId, String userId, String role) {
  return ApiCall('PATCH', '/workspaces/$workspaceId/members/$userId', {'role': role});
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
  return ApiCall('POST', '/prds/$id/revisions', note == null ? null : {'note': note});
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
    'ownerId': ownerId,
  });
}

ApiCall deleteProject(String id) => ApiCall('DELETE', '/projects/$id');

ApiCall patchProjectUpdate(String updateId, String content) {
  return ApiCall('PATCH', '/updates/$updateId', {'content': content});
}

ApiCall deleteProjectUpdate(String updateId) => ApiCall('DELETE', '/updates/$updateId');

ApiCall patchRoadmapNode(String nodeId, {String? title, String? description, String? status}) {
  return ApiCall('PATCH', '/roadmap/$nodeId', {
    if (title != null) 'title': title,
    if (description != null) 'description': description,
    if (status != null) 'status': status,
  });
}

ApiCall moveRoadmapNode(String nodeId, {String? parentId, required int position}) {
  return ApiCall('POST', '/roadmap/$nodeId/move', {
    'parentId': parentId,
    'position': position,
  });
}

ApiCall deleteRoadmapNode(String nodeId) => ApiCall('DELETE', '/roadmap/$nodeId');

ApiCall expandRoadmapNode(String nodeId) => ApiCall('POST', '/roadmap/$nodeId/expand');

ApiCall postPrdStatus(String id, String status) {
  return ApiCall('POST', '/prds/$id/status', {'status': status});
}

/// First build omits the body; a confirm retry sends `{expectedCount}`.
ApiCall postPrdBuild(String id, {int? expectedCount}) {
  return ApiCall(
    'POST',
    '/prds/$id/build',
    expectedCount == null ? null : {'expectedCount': expectedCount},
  );
}
