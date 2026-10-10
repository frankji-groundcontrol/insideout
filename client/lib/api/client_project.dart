part of 'client.dart';

/// Project, update, proposal and roadmap verbs.
mixin _ProjectVerbs on _ApiCore {
  Future<List<Project>> listProjects(String workspaceId) async {
    final res = await _dio.get('/workspaces/$workspaceId/projects');
    return _parse(
      res,
      (d) => (d as List)
          .map((e) => Project.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }

  Future<Project> createProject(
    String workspaceId,
    String title,
    String description,
  ) async {
    final res = await _dio.post(
      '/workspaces/$workspaceId/projects',
      data: {'title': title, 'description': description},
    );
    return _parse(res, (d) => Project.fromJson(d as Map<String, dynamic>));
  }

  Future<Map<String, dynamic>> getProject(String id) async {
    final res = await _dio.get('/projects/$id');
    return _parse(res, (d) => d as Map<String, dynamic>);
  }

  Future<Project> updateProject(
    String id, {
    required String title,
    required String description,
    required String status,
    String? ownerId,
  }) async {
    final res = await _exec(
      req.patchProject(
        id,
        title: title,
        description: description,
        status: status,
        ownerId: ownerId,
      ),
    );
    return _parse(res, (d) => Project.fromJson(d as Map<String, dynamic>));
  }

  Future<void> deleteProject(String id) async {
    await _parse(await _exec(req.deleteProject(id)), (d) => d);
  }

  Future<void> addUpdate(String projectId, String kind, String content) async {
    final res = await _dio.post(
      '/projects/$projectId/updates',
      data: {'kind': kind, 'content': content},
    );
    await _parse(res, (d) => d);
  }

  Future<void> editUpdate(String updateId, String content) async {
    await _parse(
      await _exec(req.patchProjectUpdate(updateId, content)),
      (d) => d,
    );
  }

  Future<void> removeUpdate(String updateId) async {
    await _parse(await _exec(req.deleteProjectUpdate(updateId)), (d) => d);
  }

  Future<Project> setRepo(String projectId, String repoUrl) async {
    final res = await _dio.put(
      '/projects/$projectId/repo',
      data: {'repoUrl': repoUrl},
    );
    return _parse(res, (d) => Project.fromJson(d as Map<String, dynamic>));
  }

  Future<Map<String, dynamic>> syncGithub(String projectId) async {
    final res = await _dio.post('/projects/$projectId/sync-github');
    return _parse(res, (d) => d as Map<String, dynamic>);
  }

  /// FEAT-03 (review 2): proposals stay proposals until a human decides.
  /// Accept defaults to apply=true so the structured items become roadmap
  /// nodes as the deciding human — the same default the CLI/MCP carries.
  Future<Map<String, dynamic>> decideProposal(
    String updateId, {
    required bool accept,
    String reason = '',
    bool apply = true,
  }) async {
    final res = await _dio.post(
      '/agent/proposals/$updateId/decision',
      data: {
        'decision': accept ? 'accepted' : 'rejected',
        'reason': reason,
        'apply': apply,
      },
    );
    return _parse(res, (d) => d as Map<String, dynamic>);
  }

  Future<List<RoadmapNode>> listRoadmap(String projectId) async {
    final res = await _dio.get('/projects/$projectId/roadmap');
    return _parse(
      res,
      (d) => (d as List)
          .map((e) => RoadmapNode.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }

  Future<RoadmapNode> createRoadmapNode(
    String projectId,
    String title, {
    String? parentId,
  }) async {
    final res = await _exec(
      req.postRoadmapNode(projectId, title, parentId: parentId),
    );
    return _parse(res, (d) => RoadmapNode.fromJson(d as Map<String, dynamic>));
  }

  Future<RoadmapNode> updateRoadmapNode(
    String nodeId, {
    String? title,
    String? description,
    String? status,
    String? deadline,
    String? acceptance,
    String? ownerId,
  }) async {
    final res = await _exec(
      req.patchRoadmapNode(
        nodeId,
        title: title,
        description: description,
        status: status,
        deadline: deadline,
        acceptance: acceptance,
        ownerId: ownerId,
      ),
    );
    return _parse(res, (d) => RoadmapNode.fromJson(d as Map<String, dynamic>));
  }

  Future<void> cursor(String projectId, double x, double y) async {
    await _dio.post('/projects/$projectId/cursor', data: {'x': x, 'y': y});
  }

  /// A node's delivery evidence, newest first (kind/detail/sourceUrl).
  Future<List<Map<String, dynamic>>> listNodeEvidence(String nodeId) async {
    final res = await _dio.get('/roadmap/$nodeId/evidence');
    final body = res.data;
    return (body is List ? body : const [])
        .map((e) => Map<String, dynamic>.from(e as Map))
        .toList();
  }

  Future<RoadmapProgress> roadmapProgress(String projectId) async {
    final res = await _dio.get('/projects/$projectId/roadmap/progress');
    return _parse(
      res,
      (d) => RoadmapProgress.fromJson(d as Map<String, dynamic>),
    );
  }

  Future<RoadmapNode> moveRoadmapNode(
    String nodeId, {
    String? parentId,
    required int position,
  }) async {
    final res = await _exec(
      req.moveRoadmapNode(nodeId, parentId: parentId, position: position),
    );
    return _parse(res, (d) => RoadmapNode.fromJson(d as Map<String, dynamic>));
  }

  Future<void> deleteRoadmapNode(String nodeId) async {
    await _parse(await _exec(req.deleteRoadmapNode(nodeId)), (d) => d);
  }

  Future<List<RoadmapNode>> expandRoadmapNode(String nodeId) async {
    final res = await _exec(req.expandRoadmapNode(nodeId));
    return _parse(
      res,
      (d) => (d as List)
          .map((e) => RoadmapNode.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }
}
