/// YYYY-MM-DD prefix of an ISO timestamp, or the input unchanged when too
/// short — display code must never RangeError on a partial date.
String isoDate(String iso) => iso.length >= 10 ? iso.substring(0, 10) : iso;

String timeAgo(String iso, String Function(String key, [Map<String, Object?>? args]) t) {
  final parsed = DateTime.tryParse(iso);
  if (parsed == null) return iso;
  final delta = DateTime.now().toUtc().difference(parsed.toUtc());
  if (delta.inMinutes < 1) return t('time.justNow');
  if (delta.inHours < 1) return t('time.minutesAgo', {'n': delta.inMinutes});
  if (delta.inHours < 24) return t('time.hoursAgo', {'n': delta.inHours});
  if (delta.inDays < 14) return t('time.daysAgo', {'n': delta.inDays});
  return isoDate(iso);
}
