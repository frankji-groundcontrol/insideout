import 'package:insideout/api/models.dart';

// The typed reads (STRC-08) parse every field the server always sends; a fake
// API names only the fields its test is about and these fill in the rest.

/// A GET /projects/{id} body: [fields] over a minimal project, parsed by the
/// real model.
ProjectDetail projectDetail(Map<String, dynamic> fields) =>
    ProjectDetail.fromJson({
      'workspaceId': 'ws-1',
      'title': '',
      'description': '',
      'status': 'active',
      'repoUrl': '',
      'createdAt': '2026-01-01T00:00:00Z',
      ...fields,
    });

/// One GET /roadmap/{id}/evidence row: [fields] over a minimal row.
NodeEvidence evidenceRow(Map<String, dynamic> fields) => NodeEvidence.fromJson({
  'id': 'e1',
  'nodeId': 'n1',
  'createdAt': '2026-01-01T00:00:00Z',
  ...fields,
});
