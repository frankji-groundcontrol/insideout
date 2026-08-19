/// Formats accepted by GET /api/v1/prds/{id}/export.
class PrdExportFormat {
  const PrdExportFormat(this.query, this.label);

  final String query;
  final String label;
}

const prdExportFormats = [
  PrdExportFormat('markdown', 'Markdown'),
  PrdExportFormat('print', 'HTML'),
];

/// Query parameters [ApiClient.exportPrd] sends. Only `markdown` and `print`
/// are valid; `html` is rejected because the Go handler 400s it.
Map<String, String> exportQueryParams(String format) {
  final allowed = {for (final f in prdExportFormats) f.query};
  if (!allowed.contains(format)) {
    throw ArgumentError.value(format, 'format', 'must be markdown or print');
  }
  return {'format': format};
}
