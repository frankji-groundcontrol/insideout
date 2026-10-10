part of 'client.dart';

/// Workspace, member and idea verbs.
mixin _WorkspaceVerbs on _ApiCore {
  Future<List<Workspace>> listWorkspaces() async {
    final res = await _dio.get('/workspaces');
    return _parse(
      res,
      (d) => (d as List)
          .map((e) => Workspace.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }

  Future<Workspace> createWorkspace(String title, String description) async {
    final res = await _dio.post(
      '/workspaces',
      data: {'title': title, 'description': description},
    );
    return _parse(res, (d) => Workspace.fromJson(d as Map<String, dynamic>));
  }

  Future<Workspace> joinWorkspace(String code) async {
    final res = await _dio.post('/workspaces/join', data: {'code': code});
    return _parse(res, (d) => Workspace.fromJson(d as Map<String, dynamic>));
  }

  Future<Workspace> getWorkspace(String id) async {
    final res = await _dio.get('/workspaces/$id');
    return _parse(res, (d) => Workspace.fromJson(d as Map<String, dynamic>));
  }

  Future<List<WorkspaceMember>> listMembers(String id) async {
    final res = await _dio.get('/workspaces/$id/members');
    return _parse(
      res,
      (d) => (d as List)
          .map((e) => WorkspaceMember.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }

  Future<Workspace> updateWorkspace(
    String id, {
    required String title,
    required String description,
    String? coverUrl,
  }) async {
    final res = await _exec(
      req.patchWorkspace(
        id,
        title: title,
        description: description,
        coverUrl: coverUrl,
      ),
    );
    return _parse(res, (d) => Workspace.fromJson(d as Map<String, dynamic>));
  }

  Future<void> deleteWorkspace(String id) async {
    await _parse(await _exec(req.deleteWorkspace(id)), (d) => d);
  }

  Future<void> updateMemberRole(
    String workspaceId,
    String userId,
    String role,
  ) async {
    await _parse(
      await _exec(req.patchMemberRole(workspaceId, userId, role)),
      (d) => d,
    );
  }

  Future<void> removeMember(String workspaceId, String userId) async {
    await _parse(await _exec(req.deleteMember(workspaceId, userId)), (d) => d);
  }

  Future<List<Idea>> listIdeas(String workspaceId) async {
    final res = await _dio.get('/workspaces/$workspaceId/ideas');
    return _parse(
      res,
      (d) => (d as List)
          .map((e) => Idea.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }

  Future<Idea> createIdea(
    String workspaceId,
    String title,
    String content,
  ) async {
    final res = await _dio.post(
      '/workspaces/$workspaceId/ideas',
      data: {'title': title, 'content': content},
    );
    return _parse(res, (d) => Idea.fromJson(d as Map<String, dynamic>));
  }

  Future<Idea> getIdea(String id) async {
    final res = await _exec(req.getIdea(id));
    return _parse(res, (d) => Idea.fromJson(d as Map<String, dynamic>));
  }

  Future<Idea> updateIdea(
    String id, {
    required String title,
    required String content,
  }) async {
    final res = await _exec(req.patchIdea(id, title: title, content: content));
    return _parse(res, (d) => Idea.fromJson(d as Map<String, dynamic>));
  }

  Future<void> dropIdea(String id) async {
    await _parse(await _exec(req.deleteIdea(id)), (d) => d);
  }

  Future<Map<String, String>> convertIdea(String ideaId) async {
    final res = await _dio.post('/ideas/$ideaId/convert');
    return _parse(
      res,
      (d) => (d as Map).map((k, v) => MapEntry(k.toString(), v.toString())),
    );
  }
}
