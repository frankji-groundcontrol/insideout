part of 'client.dart';

/// PRD, revision, commit, export and Coach-conversation verbs.
mixin _PrdVerbs on _ApiCore {
  Future<Prd> getPrd(String id) async {
    final res = await _dio.get('/prds/$id');
    return _parse(res, (d) => Prd.fromJson(d as Map<String, dynamic>));
  }

  Future<Prd> updatePrd(
    String id, {
    String? title,
    Map<String, String>? sections,
  }) async {
    final res = await _dio.patch(
      '/prds/$id',
      data: {
        if (title != null) 'title': title,
        if (sections != null) 'sections': sections,
      },
    );
    return _parse(res, (d) => Prd.fromJson(d as Map<String, dynamic>));
  }

  Future<List<PrdCommit>> listPrdCommits(String id) async {
    final res = await _dio.get('/prds/$id/commits');
    return _parse(
      res,
      (d) => (d as List)
          .map((e) => PrdCommit.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }

  Future<PrdCommit> commitPrd(
    String id, {
    required String name,
    required String audience,
    String summary = '',
    List<String>? unresolved,
    String note = '',
  }) async {
    final res = await _dio.post(
      '/prds/$id/commit',
      data: {
        'name': name,
        'primaryAudience': audience,
        'changeSummary': summary,
        'unresolved': unresolved ?? const [],
        'decisionNote': note,
      },
    );
    return _parse(res, (d) => PrdCommit.fromJson(d as Map<String, dynamic>));
  }

  Future<PrdReadiness> prdReadiness(String id) async {
    final res = await _dio.get('/prds/$id/readiness');
    return _parse(res, (d) => PrdReadiness.fromJson(d as Map<String, dynamic>));
  }

  Future<List<PrdRevision>> listRevisions(String id) async {
    final res = await _dio.get('/prds/$id/revisions');
    return _parse(
      res,
      (d) => (d as List)
          .map((e) => PrdRevision.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }

  Future<void> createRevision(String id, {String? note}) async {
    await _parse(await _exec(req.postPrdRevision(id, note: note)), (d) => d);
  }

  Future<String> exportPrd(String id, String format, {String? audience}) async {
    final params = exportQueryParams(format);
    if (audience != null) params['audience'] = audience;
    final res = await _dio.get('/prds/$id/export', queryParameters: params);
    if (res.statusCode != null && res.statusCode! >= 400) {
      throw toApiError(
        res.statusCode,
        res.data is Map<String, dynamic>
            ? res.data as Map<String, dynamic>
            : {},
      );
    }
    return res.data.toString();
  }

  Future<Conversation?> conversationForPrd(String prdId) async {
    final res = await _dio.get('/prds/$prdId/conversation');
    if (res.statusCode == 404) return null;
    return _parse(res, (d) => Conversation.fromJson(d as Map<String, dynamic>));
  }

  /// The caller's coach conversation on the PRD, started if they have none.
  Future<Conversation> startPrdConversation(String prdId) async {
    final res = await _exec(req.postPrdConversation(prdId));
    return _parse(res, (d) => Conversation.fromJson(d as Map<String, dynamic>));
  }

  /// Confirms the coach's proposed stage; the answer is the conversation
  /// with the stage written and the proposal cleared.
  Future<Conversation> confirmConversationStage(
    String conversationId,
    String stage,
  ) async {
    final res = await _exec(req.postConversationStage(conversationId, stage));
    return _parse(res, (d) => Conversation.fromJson(d as Map<String, dynamic>));
  }

  Future<Conversation> getConversation(String id) async {
    final res = await _exec(req.getConversation(id));
    return _parse(res, (d) => Conversation.fromJson(d as Map<String, dynamic>));
  }

  Future<List<ChatMessage>> listMessages(String conversationId) async {
    final res = await _dio.get('/conversations/$conversationId/messages');
    return _parse(
      res,
      (d) => (d as List)
          .map((e) => ChatMessage.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }

  Future<Prd> updatePrdStatus(String id, String status) async {
    final res = await _exec(req.postPrdStatus(id, status));
    return _parse(res, (d) => Prd.fromJson(d as Map<String, dynamic>));
  }

  /// Drafts the roadmap: `{projectId, tree}`. Nothing is written until
  /// [confirmBuildFromPrd].
  Future<Map<String, dynamic>> buildFromPrd(String id) async {
    final res = await _exec(req.postPrdBuild(id));
    return _parse(res, (d) => d as Map<String, dynamic>);
  }

  /// Persists a drafted [tree]: `{projectId, nodeCount}`.
  Future<Map<String, dynamic>> confirmBuildFromPrd(
    String id,
    Map<String, dynamic> tree, {
    int? expectedCount,
  }) async {
    final res = await _exec(
      req.postPrdBuildConfirm(id, tree, expectedCount: expectedCount),
    );
    return _parse(res, (d) => d as Map<String, dynamic>);
  }

  Future<Map<String, dynamic>> prdView(String id, String audience) async {
    final res = await _dio.get(
      '/prds/$id/view',
      queryParameters: {'audience': audience},
    );
    return _parse(res, (d) => d as Map<String, dynamic>);
  }
}
