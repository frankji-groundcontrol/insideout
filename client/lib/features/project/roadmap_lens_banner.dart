import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/models.dart';
import '../../session/appearance.dart';
import '../../theme/cobalt.dart';
import '../../theme/cobalt_theme.dart';
import '../../theme/field_card.dart';
import 'roadmap_band_scaffold.dart';
import 'roadmap_glance_lines.dart';

/// An opened node seen as its person in charge sees it: whose view this is
/// ("PRIYA NAIR'S VIEW · in charge", or "YOUR VIEW"), where the charge comes
/// from, then their first glance at the node, or one quiet line when there
/// is nothing to glance at.
class RoadmapLensBanner extends StatelessWidget {
  const RoadmapLensBanner({
    super.key,
    required this.glance,
    required this.viewerId,
    this.onOpen,
    this.full = false,
  });

  final RoadmapGlance glance;

  /// Every linked title, in a node's window.
  final bool full;

  /// The signed-in user: their own charge reads "Your view".
  final String viewerId;
  final void Function(ProgressItem item)? onOpen;

  @override
  Widget build(BuildContext context) {
    // The same padding and 1120 column as the bands below, so the edges line up.
    return Padding(
      padding: const EdgeInsets.fromLTRB(24, Cobalt.space2, 24, Cobalt.space2),
      child: RoadmapBandScaffold.capped(
        FieldCard(
          key: const Key('roadmap-lens'),
          padding: const EdgeInsets.all(Cobalt.space4),
          child: RoadmapLensBody(
            glance: glance,
            viewerId: viewerId,
            onOpen: onOpen,
            full: full,
          ),
        ),
      ),
    );
  }
}

/// The lens's lines without its frame: whose view, where the charge comes
/// from, then the glance or its quiet line.
class RoadmapLensBody extends StatelessWidget {
  const RoadmapLensBody({
    super.key,
    required this.glance,
    required this.viewerId,
    this.onOpen,
    this.full = false,
  });

  /// Show every linked title (a node's window), not two lines of them.
  final bool full;
  final RoadmapGlance glance;
  final String viewerId;
  final void Function(ProgressItem item)? onOpen;

  @override
  Widget build(BuildContext context) {
    final t = context.watch<Appearance>().t;
    final small = Theme.of(context).textTheme.bodySmall;
    final who = glance.inCharge;
    final via = who?.via;
    final notes = [
      t('roadmap.lens.inCharge'),
      if (via != null) t('roadmap.lens.via', {'title': via.title}),
      if (who?.projectOwner ?? false) t('roadmap.lens.projectOwner'),
    ];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (who != null)
          Text.rich(
            TextSpan(
              children: [
                TextSpan(
                  text:
                      (who.userId == viewerId
                              ? t('roadmap.lens.yours')
                              : t('roadmap.lens.theirs', {'name': who.name}))
                          .toUpperCase(),
                  style: eyebrow(context),
                ),
                TextSpan(text: notes.map((n) => ' · $n').join()),
              ],
            ),
            style: small,
          ),
        if (glance.isEmpty)
          Padding(
            padding: const EdgeInsets.only(top: Cobalt.space2),
            child: Text(t('roadmap.lens.quiet'), style: small),
          )
        else
          RoadmapGlanceLines(
            now: glance.now,
            waitsOn: glance.waitsOn,
            blocking: glance.blocking,
            onOpen: onOpen,
            full: full,
          ),
      ],
    );
  }
}
