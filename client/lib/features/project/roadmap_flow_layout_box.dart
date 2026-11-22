import 'package:flutter/rendering.dart';
import 'package:flutter/widgets.dart';

import 'roadmap_flow_layout.dart';

/// The Canvas flow's boxes, placed by [roadmapFlowLayout]: each child is
/// measured at its own size, then positioned so columns cross as few links
/// as they can and each box lines up with what it links to. [ids] names the
/// children in order; [columns], [links] and [groups] are over those ids.
class RoadmapFlowLayoutBox extends MultiChildRenderObjectWidget {
  const RoadmapFlowLayoutBox({
    super.key,
    required this.ids,
    required this.columns,
    required this.links,
    this.groups = const [],
    required super.children,
  });

  final List<String> ids;
  final List<List<String>> columns;
  final List<FlowLink> links;
  final List<List<String>> groups;

  @override
  RenderRoadmapFlow createRenderObject(BuildContext context) =>
      RenderRoadmapFlow(ids, columns, links, groups);

  @override
  void updateRenderObject(
    BuildContext context,
    RenderRoadmapFlow renderObject,
  ) => renderObject
    ..ids = ids
    ..columns = columns
    ..links = links
    ..groups = groups
    ..markNeedsLayout();
}

class RenderRoadmapFlow extends RenderBox
    with
        ContainerRenderObjectMixin<RenderBox, MultiChildLayoutParentData>,
        RenderBoxContainerDefaultsMixin<RenderBox, MultiChildLayoutParentData> {
  RenderRoadmapFlow(this.ids, this.columns, this.links, this.groups);

  List<String> ids;
  List<List<String>> columns;
  List<FlowLink> links;
  List<List<String>> groups;

  @override
  void setupParentData(RenderBox child) {
    if (child.parentData is! MultiChildLayoutParentData) {
      child.parentData = MultiChildLayoutParentData();
    }
  }

  @override
  void performLayout() {
    final children = <String, RenderBox>{};
    final sizes = <String, Size>{};
    var child = firstChild;
    for (final id in ids) {
      if (child == null) break;
      child.layout(const BoxConstraints(), parentUsesSize: true);
      children[id] = child;
      sizes[id] = child.size;
      child = childAfter(child);
    }
    final laid = roadmapFlowLayout(
      columns: [
        for (final column in columns) column.where(sizes.containsKey).toList(),
      ],
      links: links,
      sizes: sizes,
      groups: groups,
    );
    for (final MapEntry(key: id, value: box) in children.entries) {
      (box.parentData! as MultiChildLayoutParentData).offset =
          laid.topLeft[id] ?? Offset.zero;
    }
    size = constraints.constrain(laid.size);
  }

  @override
  void paint(PaintingContext context, Offset offset) =>
      defaultPaint(context, offset);

  @override
  bool hitTestChildren(BoxHitTestResult result, {required Offset position}) =>
      defaultHitTestChildren(result, position: position);
}
