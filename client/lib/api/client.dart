import 'dart:convert';

import 'package:dio/dio.dart';

import 'errors.dart';
import 'export_format.dart';
import 'models.dart';
import 'requests.dart' as req;
import 'sse.dart';
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

  Future<UserProfile> register(String email, String password, String username) async {
    final res = await _dio.post('/auth/register', data: {
      'email': email,
      'password': password,
      'username': username,
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

  Future<List<PrdRevision>> listRevisions(String id) async {
    final res = await _dio.get('/prds/$id/revisions');
    return _parse(res, (d) => (d as List).map((e) => PrdRevision.fromJson(e as Map<String, dynamic>)).toList());
  }

  Future<void> createRevision(String id, {String? note}) async {
    await _parse(await _exec(req.postPrdRevision(id, note: note)), (d) => d);
  }

  Future<String> exportPrd(String id, String format) async {
    final res = await _dio.get('/prds/$id/export', queryParameters: exportQueryParams(format));
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

  Future<List<RoadmapNode>> listRoadmap(String projectId) async {
    final res = await _dio.get('/projects/$projectId/roadmap');
    return _parse(res, (d) => (d as List).map((e) => RoadmapNode.fromJson(e as Map<String, dynamic>)).toList());
  }

  Future<RoadmapNode> createRoadmapNode(String projectId, String title, {String? parentId}) async {
    final res = await _exec(req.postRoadmapNode(projectId, title, parentId: parentId));
    return _parse(res, (d) => RoadmapNode.fromJson(d as Map<String, dynamic>));
  }

  Future<RoadmapNode> updateRoadmapNode(String nodeId, {String? title, String? description, String? status}) async {
    final res = await _exec(req.patchRoadmapNode(nodeId, title: title, description: description, status: status));
    return _parse(res, (d) => RoadmapNode.fromJson(d as Map<String, dynamic>));
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
}
