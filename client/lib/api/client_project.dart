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
    await _parse(
      await _exec(req.postProjectUpdate(projectId, kind, content)),
      (d) => d,
    );
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

  /// The updates older than [before], a nextCursor from the project or an
  /// earlier page (rv5-PARB-07).
  Future<UpdatesPage> listProjectUpdates(
    String projectId, {
    String? before,
  }) async {
    final res = await _exec(req.getProjectUpdates(projectId, before: before));
    return _parse(res, (d) {
      final m = d as Map<String, dynamic>;
      return (
        updates: (m['updates'] as List)
            .map((e) => ProjectUpdate.fromJson(e as Map<String, dynamic>))
            .toList(),
        nextCursor: m['nextCursor'] as String?,
      );
    });
  }

  Future<Project> setRepo(String projectId, String repoUrl) async {
    final res = await _exec(req.putProjectRepo(projectId, repoUrl));
    return _parse(res, (d) => Project.fromJson(d as Map<String, dynamic>));
  }

  Future<Map<String, dynamic>> syncGithub(String projectId) async {
    final res = await _exec(req.postSyncGithub(projectId));
    return _parse(res, (d) => d as Map<String, dynamic>);
  }

  /// The insideout.yaml guide generated from the roadmap (PARB-11).
  Future<String> getGuide(String projectId) async {
    final res = await _exec(req.getProjectGuide(projectId));
    return _parse(res, (d) => d.toString());
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
    final res = await _exec(
      req.postProposalDecision(
        updateId,
        accept: accept,
        reason: reason,
        apply: apply,
      ),
    );
    return _parse(res, (d) => d as Map<String, dynamic>);
  }

  Future<List<RoadmapNode>> listRoadmap(String projectId) async {
    final res = await _exec(req.getRoadmap(projectId));
    return _parse(
      res,
      (d) => (d as List)
          .map((e) => RoadmapNode.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }

  /// [rootId] and the nodes below it: the node opened as its own tree.
  Future<List<RoadmapNode>> listRoadmapSubtree(
    String projectId,
    String rootId,
  ) async {
    final res = await _exec(req.getRoadmap(projectId, rootId: rootId));
    return _parse(
      res,
      (d) => (d as List)
          .map((e) => RoadmapNode.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }

  /// One node with its ancestors (root first) and its project.
  Future<RoadmapNodeRead> readRoadmapNode(String nodeId) async {
    final res = await _exec(req.getRoadmapNode(nodeId));
    return _parse(
      res,
      (d) => RoadmapNodeRead.fromJson(d as Map<String, dynamic>),
    );
  }

  Future<RoadmapNode> delegateRoadmapNode(
    String nodeId,
    String toUserId, {
    String note = '',
  }) async {
    final res = await _exec(req.delegateRoadmapNode(nodeId, toUserId, note));
    return _parse(res, (d) => RoadmapNode.fromJson(d as Map<String, dynamic>));
  }

  Future<RoadmapNode> handBackRoadmapNode(
    String nodeId, {
    String note = '',
  }) async {
    final res = await _exec(req.handBackRoadmapNode(nodeId, note));
    return _parse(res, (d) => RoadmapNode.fromJson(d as Map<String, dynamic>));
  }

  /// Who held the node, newest first.
  Future<List<NodeAssignment>> listNodeAssignments(String nodeId) async {
    final res = await _exec(req.getNodeAssignments(nodeId));
    return _parse(
      res,
      (d) => (d as List)
          .map((e) => NodeAssignment.fromJson(e as Map<String, dynamic>))
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
    String? completionReason,
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
        completionReason: completionReason,
      ),
    );
    return _parse(res, (d) => RoadmapNode.fromJson(d as Map<String, dynamic>));
  }

  /// The server keys a cursor by [clientId], as it keys the presence stream
  /// opened with the same id: send that one, or the two never meet.
  Future<void> cursor(
    String projectId,
    double x,
    double y, {
    required String clientId,
  }) async {
    await _exec(req.postCursor(projectId, x, y, clientId: clientId));
  }

  /// A node's delivery evidence, newest first (kind/detail/sourceUrl).
  Future<List<Map<String, dynamic>>> listNodeEvidence(String nodeId) async {
    final res = await _exec(req.getNodeEvidence(nodeId));
    final body = res.data;
    return (body is List ? body : const [])
        .map((e) => Map<String, dynamic>.from(e as Map))
        .toList();
  }

  /// Records one human-entered validation observation on a leaf: what
  /// was observed, its source, and the date it was seen (FEAT-04).
  Future<void> addValidationObservation(
    String nodeId, {
    required String detail,
    String source = '',
    required DateTime observedAt,
  }) async {
    final res = await _exec(
      req.postNodeEvidence(
        nodeId,
        detail: detail,
        source: source,
        observedAt: observedAt,
      ),
    );
    await _parse(res, (d) => d);
  }

  /// The human confirm into 'validated': a done node plus the reason it
  /// counts as validated. The plain status PATCH cannot make this move.
  Future<RoadmapNode> validateRoadmapNode(String nodeId, String reason) async {
    final res = await _exec(req.postValidate(nodeId, reason));
    return _parse(res, (d) => RoadmapNode.fromJson(d as Map<String, dynamic>));
  }

  Future<RoadmapProgress> roadmapProgress(String projectId) async {
    final res = await _exec(req.getRoadmapProgress(projectId));
    return _parse(
      res,
      (d) => RoadmapProgress.fromJson(d as Map<String, dynamic>),
    );
  }

  /// Progress over the leaves below [rootId] only.
  Future<RoadmapProgress> roadmapSubtreeProgress(
    String projectId,
    String rootId,
  ) async {
    final res = await _exec(req.getRoadmapProgress(projectId, rootId: rootId));
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

  Future<void> deleteRoadmapNode(String nodeId, {bool force = false}) async {
    await _parse(
      await _exec(req.deleteRoadmapNode(nodeId, force: force)),
      (d) => d,
    );
  }

  /// Asks the AI to draft sub-tasks for a node. The server writes nothing: the
  /// 200 body is `{children: [...]}`. Show the draft, and send what the person
  /// keeps to [confirmExpandRoadmapNode], the only call that creates them.
  Future<List<DraftSubtask>> expandRoadmapNode(String nodeId) async {
    final res = await _exec(req.expandRoadmapNode(nodeId));
    return _parse(
      res,
      (d) => ((d as Map<String, dynamic>)['children'] as List? ?? const [])
          .map((e) => DraftSubtask.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }

  /// Creates the confirmed [draft] under the node; the server answers 201 with
  /// the created nodes in order.
  Future<List<RoadmapNode>> confirmExpandRoadmapNode(
    String nodeId,
    List<DraftSubtask> draft,
  ) async {
    final res = await _exec(
      req.confirmExpandRoadmapNode(nodeId, [for (final c in draft) c.toJson()]),
    );
    return _parse(
      res,
      (d) => (d as List)
          .map((e) => RoadmapNode.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }
}
