/// Removes [keys] from the query string of [url].
///
/// Other query keys and the fragment stay. When nothing is removed, [url]
/// is returned unchanged. No Flutter binding.
String urlWithoutQueryKeys(String url, Iterable<String> keys) {
  final drop = Set<String>.of(keys);
  if (drop.isEmpty) return url;
  final uri = Uri.parse(url);
  final kept = <String, List<String>>{};
  var removed = false;
  for (final entry in uri.queryParametersAll.entries) {
    if (drop.contains(entry.key)) {
      removed = true;
    } else {
      kept[entry.key] = entry.value;
    }
  }
  if (!removed) return url;
  final rewritten = uri
      .replace(
        queryParameters: <String, dynamic>{
          for (final entry in kept.entries) entry.key: entry.value,
        },
      )
      .toString();
  return _withoutEmptyQuery(rewritten);
}

/// [Uri.replace] keeps an empty query as `?`, which we do not want in history.
String _withoutEmptyQuery(String url) {
  final hash = url.indexOf('#');
  final head = hash == -1 ? url : url.substring(0, hash);
  if (!head.endsWith('?')) return url;
  final tail = hash == -1 ? '' : url.substring(hash);
  return head.substring(0, head.length - 1) + tail;
}
