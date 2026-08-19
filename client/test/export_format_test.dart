import 'package:flutter_test/flutter_test.dart';
import 'package:insideout/api/export_format.dart';

void main() {
  test('exportQueryParams sends markdown or print only', () {
    expect(exportQueryParams('markdown'), {'format': 'markdown'});
    expect(exportQueryParams('print'), {'format': 'print'});
  });

  test('exportQueryParams rejects html which the Go API does not accept', () {
    expect(() => exportQueryParams('html'), throwsArgumentError);
  });

  test('ExportPage segments use the same query values the API helper allows', () {
    expect(
      prdExportFormats.map((f) => f.query).toSet(),
      {'markdown', 'print'},
    );
    for (final option in prdExportFormats) {
      expect(exportQueryParams(option.query)['format'], option.query);
    }
  });
}
