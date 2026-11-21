part of 'client.dart';

/// Project, update and proposal verbs; the roadmap's live in client_roadmap.dart.
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

  Future<ProjectDetail> getProject(String id) async {
    final res = await _dio.get('/projects/$id');
    return _parse(
      res,
      (d) => ProjectDetail.fromJson(d as Map<String, dynamic>),
    );
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
  /// nodes as the deciding human — the same default the CLI's `proposal
  /// decide` carries (MCP has no decision tool: its route refuses an agent
  /// token).
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
}
