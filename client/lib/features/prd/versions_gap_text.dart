import '../../api/models.dart';
import '../../session/appearance.dart';

/// The gap's reason in the reader's language. The server sends one fixed
/// English sentence per audience and section (readiness.go); a pair the
/// client has no translation for keeps the server's text.
String gapReason(Appearance l10n, String audience, ReadinessGap gap) {
  final key = gap.priority == 'validate_later'
      ? 'versions.gap.later'
      : 'versions.gap.$audience.${gap.section}';
  final text = l10n.t(key);
  return text == key ? gap.reason : text;
}

const _carriedSuffix = ' (carried as open question)';

/// A carried open item in the reader's language. The server stores the gaps a
/// commit carries as `<section>: <reason> (carried as open question)`
/// (readiness.go), so those read like the gaps they came from. Anything else,
/// such as an item the caller named, is shown as stored.
String carriedItem(Appearance l10n, String audience, String item) {
  final colon = item.indexOf(': ');
  if (colon <= 0 || !item.endsWith(_carriedSuffix)) return item;
  final section = item.substring(0, colon);
  final key = 'versions.gap.$audience.$section';
  final text = l10n.t(key);
  if (text == key) return item;
  return '${l10n.t('prd.sections.$section')} — $text';
}
