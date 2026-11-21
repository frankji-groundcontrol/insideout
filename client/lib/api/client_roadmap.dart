part of 'client.dart';

/// Roadmap node, evidence, validation, progress and expand verbs.
mixin _RoadmapVerbs on _ApiCore {
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
  Future<List<NodeEvidence>> listNodeEvidence(String nodeId) async {
    final res = await _exec(req.getNodeEvidence(nodeId));
    // A refused read (4xx) reads as no evidence, as it always has.
    final body = res.data;
    return (body is List ? body : const [])
        .map((e) => NodeEvidence.fromJson(e as Map<String, dynamic>))
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
