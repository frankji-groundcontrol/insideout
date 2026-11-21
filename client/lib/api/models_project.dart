class Project {
  Project({
    required this.id,
    required this.workspaceId,
    required this.title,
    required this.description,
    required this.status,
    required this.repoUrl,
    required this.createdAt,
    required this.flagged,
    this.latestUpdateKind,
    this.latestUpdateContent,
    this.latestUpdateAt,
    this.ownerId,
    this.prdId,
    this.canWritePlan = false,
  });

  final String id;
  final String workspaceId;
  final String title;
  final String description;
  final String status;
  final String repoUrl;
  final String createdAt;
  final bool flagged;
  final String? latestUpdateKind;
  final String? latestUpdateContent;
  final String? latestUpdateAt;

  /// The project owner: with workspace admins, the one who writes its plan.
  final String? ownerId;

  /// The PRD the project was built from; null for one made by hand.
  final String? prdId;

  /// Whether the caller may change the plan (owner or workspace admin).
  final bool canWritePlan;

  factory Project.fromJson(Map<String, dynamic> j) => Project(
    id: j['id'] as String,
    workspaceId: j['workspaceId'] as String,
    title: j['title'] as String,
    description: (j['description'] as String?) ?? '',
    status: j['status'] as String,
    repoUrl: (j['repoUrl'] as String?) ?? '',
    createdAt: j['createdAt'] as String,
    flagged: (j['flagged'] as bool?) ?? false,
    latestUpdateKind: j['latestUpdateKind'] as String?,
    latestUpdateContent: j['latestUpdateContent'] as String?,
    latestUpdateAt: j['latestUpdateAt'] as String?,
    ownerId: j['ownerId'] as String?,
    prdId: j['prdId'] as String?,
    canWritePlan: (j['canWritePlan'] as bool?) ?? false,
  );
}

/// One row of GET /workspaces/{id}/prds: every PRD of the workspace, so a
/// teammate or the approving admin can find one they did not write.
class WorkspacePrd {
  WorkspacePrd({
    required this.id,
    required this.title,
    required this.status,
    required this.authorName,
    required this.updatedAt,
    this.projectId,
  });

  final String id;
  final String title;
  final String status;
  final String authorName;
  final String updatedAt;
  final String? projectId;

  factory WorkspacePrd.fromJson(Map<String, dynamic> j) => WorkspacePrd(
    id: j['id'] as String,
    title: j['title'] as String,
    status: j['status'] as String,
    authorName: (j['authorName'] as String?) ?? '',
    updatedAt: j['updatedAt'] as String,
    projectId: j['projectId'] as String?,
  );
}

class ProposalItem {
  ProposalItem({
    required this.action,
    required this.title,
    this.parentHint = '',
  });

  final String action;
  final String title;
  final String parentHint;

  factory ProposalItem.fromJson(Map<String, dynamic> j) => ProposalItem(
    action: (j['action'] as String?) ?? 'add_node',
    title: j['title'] as String,
    parentHint: (j['parentHint'] as String?) ?? '',
  );
}

class ProjectUpdate {
  ProjectUpdate({
    required this.id,
    required this.kind,
    required this.content,
    required this.createdAt,
    required this.authorId,
    this.items = const [],
    this.decision = '',
  });

  final String id;
  final String kind;
  final String content;
  final String createdAt;
  final String authorId;
  final List<ProposalItem> items;

  /// "accepted" or "rejected" once a human has decided a proposal.
  final String decision;

  factory ProjectUpdate.fromJson(Map<String, dynamic> j) => ProjectUpdate(
    id: j['id'] as String,
    kind: j['kind'] as String,
    content: j['content'] as String,
    createdAt: j['createdAt'] as String,
    authorId: j['authorId'] as String,
    items: ((j['items'] as List?) ?? const [])
        .map((e) => ProposalItem.fromJson(e as Map<String, dynamic>))
        .toList(),
    decision: j['decision'] as String? ?? '',
  );
}

/// One page of a project's updates and the cursor of the next, older one.
typedef UpdatesPage = ({List<ProjectUpdate> updates, String? nextCursor});

class Idea {
  Idea({
    required this.id,
    required this.workspaceId,
    required this.title,
    required this.content,
    required this.status,
    required this.createdAt,
    this.prdId,
  });

  final String id;
  final String workspaceId;
  final String title;
  final String content;
  final String status;
  final String createdAt;
  final String? prdId;

  factory Idea.fromJson(Map<String, dynamic> j) => Idea(
    id: j['id'] as String,
    workspaceId: j['workspaceId'] as String,
    title: j['title'] as String,
    content: (j['content'] as String?) ?? '',
    status: j['status'] as String,
    createdAt: j['createdAt'] as String,
    prdId: j['prdId'] as String?,
  );
}
