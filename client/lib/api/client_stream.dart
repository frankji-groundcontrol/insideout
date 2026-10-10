part of 'client.dart';

/// Server-sent-event verbs: Coach replies, canvas presence, roadmap progress.
mixin _StreamVerbs on _ApiCore {
  Future<void> sendCoach(
    String conversationId,
    String content,
    CoachHandlers handlers,
  ) async {
    final token = await _readAccess();
    final open = coachSseOpen(
      baseUrl: baseUrl,
      conversationId: conversationId,
      content: content,
      accessToken: token,
    );
    Stream<List<int>> chunks;
    if (kIsWeb) {
      // USER-10: this raw fetch bypasses the dio interceptor that owns the
      // 401 refresh; the opener re-reads the access token, so the
      // post-refresh retry carries the rotated bearer automatically.
      chunks = coachSseWithRefresh(
        () async {
          final o = coachSseOpen(
            baseUrl: baseUrl,
            conversationId: conversationId,
            content: content,
            accessToken: await _readAccess(),
          );
          return fetchSseAttempt(
            o.url,
            o.headers,
            method: o.method,
            body: o.body,
          );
        },
        _refreshSingleFlight,
        _clearTokens,
      );
    } else {
      final res = await _dio.post<ResponseBody>(
        '/conversations/$conversationId/messages',
        data: {'content': content},
        options: Options(
          responseType: ResponseType.stream,
          headers: open.headers,
        ),
      );
      if (res.statusCode != null && res.statusCode! >= 400) {
        throw ApiException('coach failed', status: res.statusCode);
      }
      chunks = res.data!.stream;
    }
    await readCoachSse(chunks, handlers, idleTimeout: coachSseIdleTimeout);
  }

  /// Subscribes to the project's live presence (canvas who-is-here).
  /// Every snapshot replace-calls [onChange]; the stream ends when the
  /// caller cancels the dio request (dio has no per-request cancel
  /// here, so the page closes by letting it die with dispose).
  Future<void> presenceStream(
    String projectId,
    String clientId,
    void onChange(List<PresenceEntry> entries), {
    void Function(String sessionId, String name, double x, double y)? onCursor,
    Future<void>? stop,
  }) async {
    final token = await _readAccess();
    final headers = {
      if (token != null) 'Authorization': 'Bearer $token',
      'Accept': 'text/event-stream',
    };
    Stream<List<int>> chunks;
    if (kIsWeb) {
      chunks = fetchSseStream(
        '$baseUrl/projects/$projectId/presence/stream?client=${Uri.encodeQueryComponent(clientId)}',
        headers,
      );
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
    await for (final text in decodeSseBytes(chunks)) {
      buffer += text;
      final parsed = parseSseBuffer(buffer);
      buffer = parsed.rest;
      for (final frame in parsed.frames) {
        if (frame.event == 'presence') {
          final list = jsonDecode(frame.data);
          onChange(
            (list is List ? list : const [])
                .map((e) => PresenceEntry.fromJson(e as Map<String, dynamic>))
                .toList(),
          );
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

  /// Subscribes to live progress snapshots. Every `progress` frame
  /// replace-calls [onProgress]; the stream ends when the page disposes.
  Future<void> progressStream(
    String projectId,
    void Function(RoadmapProgress progress) onProgress, {
    Future<void>? stop,
  }) async {
    final token = await _readAccess();
    final headers = {
      if (token != null) 'Authorization': 'Bearer $token',
      'Accept': 'text/event-stream',
    };
    Stream<List<int>> chunks;
    if (kIsWeb) {
      chunks = fetchSseStream(
        '$baseUrl/projects/$projectId/roadmap/progress/stream',
        headers,
      );
    } else {
      final res = await _dio.get<ResponseBody>(
        '/projects/$projectId/roadmap/progress/stream',
        options: Options(responseType: ResponseType.stream, headers: headers),
      );
      if (res.statusCode != null && res.statusCode! >= 400) {
        throw ApiException('progress stream failed', status: res.statusCode);
      }
      chunks = res.data!.stream;
    }
    await readProgressSse(chunks, onProgress);
  }
}
