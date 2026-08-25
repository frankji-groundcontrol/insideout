import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

import 'errors.dart';
import 'export_format.dart';
import 'models.dart';
import 'requests.dart' as req;
import 'sse.dart';
import 'sse_transport_stub.dart'
    if (dart.library.ui_web) 'sse_transport_web.dart';
import '../session/token_store.dart';

typedef TokenReader = Future<String?> Function();
typedef TokenWriter = Future<void> Function(String access, String refresh);
typedef TokenClearer = Future<void> Function();

class ApiClient {
  ApiClient({
    required this.baseUrl,
    required TokenReader readAccess,
    required TokenReader readRefresh,
    required TokenWriter writeTokens,
    required TokenClearer clearTokens,
  }) : _readAccess = readAccess,
       _readRefresh = readRefresh,
       _writeTokens = writeTokens,
       _clearTokens = clearTokens {
    _dio = Dio(BaseOptions(
      baseUrl: baseUrl,
      headers: {'Content-Type': 'application/json'},
      validateStatus: (s) => s != null && s < 500,
    ));
    _dio.interceptors.add(InterceptorsWrapper(
      onRequest: (options, handler) async {
        final token = await _readAccess();
        if (token != null && token.isNotEmpty) {
          options.headers['Authorization'] = 'Bearer $token';
        }
        handler.next(options);
      },
    ));
  }

  final String baseUrl;
  final TokenReader _readAccess;
  final TokenReader _readRefresh;
  final TokenWriter _writeTokens;
  final TokenClearer _clearTokens;
  late final Dio _dio;

  Future<T> _parse<T>(Response res, T Function(dynamic) map) async {
    if (res.statusCode != null && res.statusCode! >= 400) {
      final body = res.data is Map<String, dynamic>
          ? res.data as Map<String, dynamic>
          : <String, dynamic>{'error': res.statusMessage};
      throw toApiError(res.statusCode, body);
    }
    return map(res.data);
  }

  Future<Response<dynamic>> _exec(req.ApiCall call) {
    return _dio.request<dynamic>(call.path, data: call.body, options: Options(method: call.method));
  }

  Future<UserProfile> register(String email, String password, String username, {String inviteCode = ''}) async {
    final res = await _dio.post('/auth/register', data: {
      'email': email,
      'password': password,
      'username': username,
      if (inviteCode.isNotEmpty) 'inviteCode': inviteCode,
    });
    return _parse(res, (d) => UserProfile.fromJson(d as Map<String, dynamic>));
  }

  Future<UserProfile> login(String email, String password) async {
    final res = await _dio.post('/auth/login', data: {
      'email': email,
      'password': password,
    });
    return _parse(res, (d) => UserProfile.fromJson(d as Map<String, dynamic>));
  }

  /// Requests a magic-link email for [email]. Always resolves when the
  /// server accepts the request — the server answers 200 regardless of
  /// whether the account exists.
  Future<void> requestMagicLink(String email) async {
    final res = await _dio.post('/auth/magic-link', data: {'email': email});
    await _parse(res, (d) => d);
  }

  /// Exchanges an emailed magic-link token for a full session (same shape
  /// as login). Throws with status 401 when the link is invalid, expired,
  /// or already used.
  Future<UserProfile> consumeMagicLink(String token) async {
    final res = await _dio.post('/auth/magic-link/consume', data: {'token': token});
    return _parse(res, (d) => UserProfile.fromJson(d as Map<String, dynamic>));
  }

  /// Requests a 6-digit sign-in code email for [email]. Always resolves
  /// when the server accepts the request (uniform 200 contract).
  Future<void> requestLoginCode(String email) async {
    final res = await _dio.post('/auth/login-code', data: {'email': email});
    await _parse(res, (d) => d);
  }

  /// Exchanges an emailed sign-in code for a full session (same shape as
  /// login). Throws with status 401 when the code is wrong, expired, or
  /// exhausted by failed attempts.
  Future<UserProfile> verifyLoginCode(String email, String code) async {
    final res = await _dio.post('/auth/login-code/verify', data: {
      'email': email,
      'code': code,
    });
    return _parse(res, (d) => UserProfile.fromJson(d as Map<String, dynamic>));
  }

  /// Requests a password-reset email for [email] (uniform 200 contract).
  Future<void> forgotPassword(String email) async {
    final res = await _dio.post('/auth/forgot-password', data: {'email': email});
    await _parse(res, (d) => d);
  }
  /// Sets a new password with an emailed reset token. On success every
  /// existing session of that account has been revoked server-side, so the
  /// next step is always a fresh login.
  Future<void> resetPassword(String token, String password) async {
    final res = await _dio.post('/auth/reset-password', data: {
      'token': token,
      'password': password,
    });
    await _parse(res, (d) => d);
  }

  /// Public signup policy — the register form needs to know whether an
  /// invitation code field should be shown. Failing open to
  /// invitation-only would be wrong; the caller decides the fallback.
  Future<String> authConfig() async {
    final res = await _dio.get('/auth/config');
    final data = await _parse(res, (d) => d as Map<String, dynamic>);
    return data['signupMode'] as String? ?? 'invitation_only';
  }

  /// Admin: mint an email-bound invitation. Returns the one-time code and
  /// signup URL (the only place the code ever appears).
  Future<Map<String, dynamic>> createInvitation(String email) async {
    final res = await _dio.post('/admin/invitations', data: {'email': email});
    return _parse(res, (d) => d as Map<String, dynamic>);
  }

  /// Admin: invitation ledger without codes.
  Future<List<Map<String, dynamic>>> listInvitations() async {
    final res = await _dio.get('/admin/invitations');
    final data = await _parse(res, (d) => d);
    return ((data as List?) ?? const []).cast<Map<String, dynamic>>();
  }

  /// Admin: revoke a still-usable invitation.
  Future<void> revokeInvitation(String id) async {
    final res = await _dio.post('/admin/invitations/$id/revoke', data: {});
    await _parse(res, (d) => d);
  }

  /// Admin: switch registration policy (invitation_only | open | closed).
  Future<void> setSignupMode(String mode) async {
    final res = await _dio.put('/admin/settings', data: {'signupMode': mode});
    await _parse(res, (d) => d);
  }

  Future<void> logout() async {
    final refresh = await _readRefresh();
    await _dio.post('/auth/logout', data: {'refreshToken': refresh ?? ''});
    await _clearTokens();
  }

  Future<void> refresh() async {
    final refresh = await _readRefresh();
    if (refresh == null || refresh.isEmpty) {
      throw ApiException('no refresh token', status: 401);
    }
    final res = await _dio.post('/auth/refresh', data: {'refreshToken': refresh});
    final data = await _parse(res, (d) => d as Map<String, dynamic>);
    final pair = tokensFromRefreshBody(data);
    await _writeTokens(pair.access, pair.refresh);
  }

  Future<UserProfile> me() async {
    final res = await _dio.get('/me');
    return _parse(res, (d) => UserProfile.fromJson(d as Map<String, dynamic>));
  }

  Future<UserProfile> updateMe({
    required String username,
    required String bio,
    required List<String> keywords,
  }) async {
    final res = await _dio.patch('/me', data: {
      'username': username,
      'bio': bio,
      'keywords': keywords,
    });
    return _parse(res, (d) => UserProfile.fromJson(d as Map<String, dynamic>));
  }

  Future<List<Workspace>> listWorkspaces() async {
    final res = await _dio.get('/workspaces');
    return _parse(res, (d) => (d as List).map((e) => Workspace.fromJson(e as Map<String, dynamic>)).toList());
  }

  Future<Workspace> createWorkspace(String title, String description) async {
    final res = await _dio.post('/workspaces', data: {'title': title, 'description': description});
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
    return _parse(res, (d) => (d as List).map((e) => WorkspaceMember.fromJson(e as Map<String, dynamic>)).toList());
  }

  Future<Workspace> updateWorkspace(String id, {required String title, required String description, String? coverUrl}) async {
    final res = await _exec(req.patchWorkspace(id, title: title, description: description, coverUrl: coverUrl));
    return _parse(res, (d) => Workspace.fromJson(d as Map<String, dynamic>));
  }

  Future<void> deleteWorkspace(String id) async {
    await _parse(await _exec(req.deleteWorkspace(id)), (d) => d);
  }

  Future<void> updateMemberRole(String workspaceId, String userId, String role) async {
    await _parse(await _exec(req.patchMemberRole(workspaceId, userId, role)), (d) => d);
  }

  Future<void> removeMember(String workspaceId, String userId) async {
    await _parse(await _exec(req.deleteMember(workspaceId, userId)), (d) => d);
  }

  Future<List<Project>> listProjects(String workspaceId) async {
    final res = await _dio.get('/workspaces/$workspaceId/projects');
    return _parse(res, (d) => (d as List).map((e) => Project.fromJson(e as Map<String, dynamic>)).toList());
  }

  Future<Project> createProject(String workspaceId, String title, String description) async {
    final res = await _dio.post('/workspaces/$workspaceId/projects', data: {
      'title': title,
      'description': description,
    });
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
    final res = await _exec(req.patchProject(id, title: title, description: description, status: status, ownerId: ownerId));
    return _parse(res, (d) => Project.fromJson(d as Map<String, dynamic>));
  }

  Future<void> deleteProject(String id) async {
    await _parse(await _exec(req.deleteProject(id)), (d) => d);
  }

  Future<void> addUpdate(String projectId, String kind, String content) async {
    final res = await _dio.post('/projects/$projectId/updates', data: {'kind': kind, 'content': content});
    await _parse(res, (d) => d);
  }

  Future<void> editUpdate(String updateId, String content) async {
    await _parse(await _exec(req.patchProjectUpdate(updateId, content)), (d) => d);
  }

  Future<void> removeUpdate(String updateId) async {
    await _parse(await _exec(req.deleteProjectUpdate(updateId)), (d) => d);
  }

  Future<Project> setRepo(String projectId, String repoUrl) async {
    final res = await _dio.put('/projects/$projectId/repo', data: {'repoUrl': repoUrl});
    return _parse(res, (d) => Project.fromJson(d as Map<String, dynamic>));
  }

  Future<Map<String, dynamic>> syncGithub(String projectId) async {
    final res = await _dio.post('/projects/$projectId/sync-github');
    return _parse(res, (d) => d as Map<String, dynamic>);
  }

  Future<List<Idea>> listIdeas(String workspaceId) async {
    final res = await _dio.get('/workspaces/$workspaceId/ideas');
    return _parse(res, (d) => (d as List).map((e) => Idea.fromJson(e as Map<String, dynamic>)).toList());
  }

  Future<Idea> createIdea(String workspaceId, String title, String content) async {
    final res = await _dio.post('/workspaces/$workspaceId/ideas', data: {'title': title, 'content': content});
    return _parse(res, (d) => Idea.fromJson(d as Map<String, dynamic>));
  }

  Future<Idea> getIdea(String id) async {
    final res = await _exec(req.getIdea(id));
    return _parse(res, (d) => Idea.fromJson(d as Map<String, dynamic>));
  }

  Future<Idea> updateIdea(String id, {required String title, required String content}) async {
    final res = await _exec(req.patchIdea(id, title: title, content: content));
    return _parse(res, (d) => Idea.fromJson(d as Map<String, dynamic>));
  }

  Future<void> dropIdea(String id) async {
    await _parse(await _exec(req.deleteIdea(id)), (d) => d);
  }

  Future<Map<String, String>> convertIdea(String ideaId) async {
    final res = await _dio.post('/ideas/$ideaId/convert');
    return _parse(res, (d) => (d as Map).map((k, v) => MapEntry(k.toString(), v.toString())));
  }

  Future<Prd> getPrd(String id) async {
    final res = await _dio.get('/prds/$id');
    return _parse(res, (d) => Prd.fromJson(d as Map<String, dynamic>));
  }

  Future<Prd> updatePrd(String id, {String? title, Map<String, String>? sections}) async {
    final res = await _dio.patch('/prds/$id', data: {
      if (title != null) 'title': title,
      if (sections != null) 'sections': sections,
    });
    return _parse(res, (d) => Prd.fromJson(d as Map<String, dynamic>));
  }

  Future<List<PrdCommit>> listPrdCommits(String id) async {
    final res = await _dio.get('/prds/$id/commits');
    return _parse(res, (d) => (d as List).map((e) => PrdCommit.fromJson(e as Map<String, dynamic>)).toList());
  }

  Future<PrdCommit> commitPrd(String id, {required String name, required String audience, String summary = '', List<String>? unresolved, String note = ''}) async {
    final res = await _dio.post('/prds/$id/commit', data: {
      'name': name,
      'primaryAudience': audience,
      'changeSummary': summary,
      'unresolved': unresolved ?? const [],
      'decisionNote': note,
    });
    return _parse(res, (d) => PrdCommit.fromJson(d as Map<String, dynamic>));
  }

  Future<PrdReadiness> prdReadiness(String id) async {
    final res = await _dio.get('/prds/$id/readiness');
    return _parse(res, (d) => PrdReadiness.fromJson(d as Map<String, dynamic>));
  }

  Future<List<PrdRevision>> listRevisions(String id) async {
    final res = await _dio.get('/prds/$id/revisions');
    return _parse(res, (d) => (d as List).map((e) => PrdRevision.fromJson(e as Map<String, dynamic>)).toList());
  }

  Future<void> createRevision(String id, {String? note}) async {
    await _parse(await _exec(req.postPrdRevision(id, note: note)), (d) => d);
  }

  Future<String> exportPrd(String id, String format, {String? audience}) async {
    final params = exportQueryParams(format);
    if (audience != null) params['audience'] = audience;
    final res = await _dio.get('/prds/$id/export', queryParameters: params);
    if (res.statusCode != null && res.statusCode! >= 400) {
      throw toApiError(res.statusCode, res.data is Map<String, dynamic> ? res.data as Map<String, dynamic> : {});
    }
    return res.data.toString();
  }

  Future<Conversation?> conversationForPrd(String prdId) async {
    final res = await _dio.get('/prds/$prdId/conversation');
    if (res.statusCode == 404) return null;
    return _parse(res, (d) => Conversation.fromJson(d as Map<String, dynamic>));
  }

  Future<Conversation> getConversation(String id) async {
    final res = await _exec(req.getConversation(id));
    return _parse(res, (d) => Conversation.fromJson(d as Map<String, dynamic>));
  }

  Future<List<ChatMessage>> listMessages(String conversationId) async {
    final res = await _dio.get('/conversations/$conversationId/messages');
    return _parse(res, (d) => (d as List).map((e) => ChatMessage.fromJson(e as Map<String, dynamic>)).toList());
  }

  Future<Prd> updatePrdStatus(String id, String status) async {
    final res = await _exec(req.postPrdStatus(id, status));
    return _parse(res, (d) => Prd.fromJson(d as Map<String, dynamic>));
  }

  Future<Map<String, dynamic>> buildFromPrd(String id, {int? expectedCount}) async {
    final res = await _exec(req.postPrdBuild(id, expectedCount: expectedCount));
    return _parse(res, (d) => d as Map<String, dynamic>);
  }

  Future<void> sendCoach(String conversationId, String content, CoachHandlers handlers) async {
    final token = await _readAccess();
    final res = await _dio.post<ResponseBody>(
      '/conversations/$conversationId/messages',
      data: {'content': content},
      options: Options(
        responseType: ResponseType.stream,
        headers: {
          if (token != null) 'Authorization': 'Bearer $token',
          'Accept': 'text/event-stream',
        },
      ),
    );
    if (res.statusCode != null && res.statusCode! >= 400) {
      throw ApiException('coach failed', status: res.statusCode);
    }
    final stream = res.data!.stream;
    var buffer = '';
    await for (final chunk in stream) {
      buffer += utf8.decode(chunk);
      final parsed = parseSseBuffer(buffer);
      buffer = parsed.rest;
      for (final frame in parsed.frames) {
        applyCoachFrame(frame, handlers);
      }
    }
  }

  /// Subscribes to the project's live presence (canvas who-is-here).
  /// Every snapshot replace-calls [onChange]; the stream ends when the
  /// caller cancels the dio request (dio has no per-request cancel
  /// here, so the page closes by letting it die with dispose).
  Future<void> presenceStream(String projectId, String clientId, void onChange(List<PresenceEntry> entries),
      {void Function(String sessionId, String name, double x, double y)? onCursor}) async {
    final token = await _readAccess();
    final headers = {
      if (token != null) 'Authorization': 'Bearer $token',
      'Accept': 'text/event-stream',
    };
    Stream<List<int>> chunks;
    if (kIsWeb) {
      chunks = fetchSseStream(
          '$baseUrl/projects/$projectId/presence/stream?client=${Uri.encodeQueryComponent(clientId)}',
          headers);
    } else {
      final res = await _dio.get<ResponseBody>(
        '/projects/$projectId/presence/stream',
        queryParameters: {'client': clientId},
        options: Options(responseType: ResponseType.stream, headers: headers),
      );
      if (res.statusCode != null && res.statusCode! >= 400) {
        throw ApiException('presence failed', status: res.statusCode);
      }
      chunks = res.data!.stream;
    }
    var buffer = '';
    await for (final text in chunks.transform(utf8.decoder)) {
      buffer += text;
      final parsed = parseSseBuffer(buffer);
      buffer = parsed.rest;
      for (final frame in parsed.frames) {
        if (frame.event == 'presence') {
          final list = jsonDecode(frame.data);
          onChange((list is List ? list : const [])
              .map((e) => PresenceEntry.fromJson(e as Map<String, dynamic>))
              .toList());
        } else if (frame.event == 'cursor' && onCursor != null) {
          final m = jsonDecode(frame.data) as Map<String, dynamic>;
          onCursor(
            m['sessionId']?.toString() ?? '',
            m['name']?.toString() ?? '',
            (m['x'] as num?)?.toDouble() ?? 0,
            (m['y'] as num?)?.toDouble() ?? 0,
          );
        }
      }
    }
  }

  Future<List<RoadmapNode>> listRoadmap(String projectId) async {
    final res = await _dio.get('/projects/$projectId/roadmap');
    return _parse(res, (d) => (d as List).map((e) => RoadmapNode.fromJson(e as Map<String, dynamic>)).toList());
  }

  Future<RoadmapNode> createRoadmapNode(String projectId, String title, {String? parentId}) async {
    final res = await _exec(req.postRoadmapNode(projectId, title, parentId: parentId));
    return _parse(res, (d) => RoadmapNode.fromJson(d as Map<String, dynamic>));
  }

  Future<RoadmapNode> updateRoadmapNode(String nodeId, {String? title, String? description, String? status, String? deadline}) async {
    final res = await _exec(req.patchRoadmapNode(nodeId, title: title, description: description, status: status, deadline: deadline));
    return _parse(res, (d) => RoadmapNode.fromJson(d as Map<String, dynamic>));
  }

  Future<void> cursor(String projectId, double x, double y) async {
    await _dio.post('/projects/$projectId/cursor', data: {'x': x, 'y': y});
  }

  Future<RoadmapProgress> roadmapProgress(String projectId) async {
    final res = await _dio.get('/projects/$projectId/roadmap/progress');
    return _parse(res, (d) => RoadmapProgress.fromJson(d as Map<String, dynamic>));
  }

  Future<RoadmapNode> moveRoadmapNode(String nodeId, {String? parentId, required int position}) async {
    final res = await _exec(req.moveRoadmapNode(nodeId, parentId: parentId, position: position));
    return _parse(res, (d) => RoadmapNode.fromJson(d as Map<String, dynamic>));
  }

  Future<void> deleteRoadmapNode(String nodeId) async {
    await _parse(await _exec(req.deleteRoadmapNode(nodeId)), (d) => d);
  }

  Future<List<RoadmapNode>> expandRoadmapNode(String nodeId) async {
    final res = await _exec(req.expandRoadmapNode(nodeId));
    return _parse(res, (d) => (d as List).map((e) => RoadmapNode.fromJson(e as Map<String, dynamic>)).toList());
  }

  Future<Map<String, dynamic>> prdView(String id, String audience) async {
    final res = await _dio.get('/prds/$id/view', queryParameters: {'audience': audience});
    return _parse(res, (d) => d as Map<String, dynamic>);
  }
}
