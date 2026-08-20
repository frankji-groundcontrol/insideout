import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../session/appearance.dart';
import '../../theme/ink_motion.dart';
import 'assembly_diagram.dart';
import 'step_peek.dart';

class AssemblyStepData {
  const AssemblyStepData({
    required this.n,
    required this.sealAsset,
    required this.titleKey,
    required this.bodyKey,
    required this.kind,
  });

  final int n;
  final String sealAsset;
  final String titleKey;
  final String bodyKey;
  final PeekKind kind;
}

class AssemblyStep extends StatelessWidget {
  const AssemblyStep({super.key, required this.data, this.flip = false});

  final AssemblyStepData data;
  final bool flip;

  @override
  Widget build(BuildContext context) {
    final l10n = context.watch<Appearance>();
    final theme = Theme.of(context);
    final wide = MediaQuery.sizeOf(context).width >= 900;
    final copy = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Image.asset(data.sealAsset, width: 40, height: 40, filterQuality: FilterQuality.medium),
            const SizedBox(width: 12),
            Text(l10n.t('landing.stepLabel', {'n': data.n}), style: theme.textTheme.bodySmall),
          ],
        ),
        const SizedBox(height: 20),
        Text(l10n.t(data.titleKey), style: theme.textTheme.headlineMedium),
        const SizedBox(height: 12),
        Text(l10n.t(data.bodyKey), style: theme.textTheme.bodyLarge),
      ],
    );
    final peek = StepPeek(kind: data.kind);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 32),
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 1100),
        child: Column(
          children: [
            InkReveal(
              whenVisible: true,
              duration: const Duration(milliseconds: 600),
              dy: 20,
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 420),
                child: AssemblyDiagram(
                  currentStep: data.n,
                  whenVisible: true,
                  semanticsLabel: l10n.t('landing.diagramAlt'),
                ),
              ),
            ),
            const SizedBox(height: 36),
            if (wide)
              Row(
                crossAxisAlignment: CrossAxisAlignment.center,
                children: [
                  Expanded(
                    child: InkReveal(
                      whenVisible: true,
                      duration: const Duration(milliseconds: 650),
                      dy: 28,
                      child: flip ? peek : copy,
                    ),
                  ),
                  const SizedBox(width: 48),
                  Expanded(
                    child: InkReveal(
                      whenVisible: true,
                      delay: const Duration(milliseconds: 80),
                      duration: const Duration(milliseconds: 650),
                      dy: 28,
                      child: flip ? copy : peek,
                    ),
                  ),
                ],
              )
            else
              Column(
                children: [
                  InkReveal(whenVisible: true, duration: const Duration(milliseconds: 650), dy: 28, child: copy),
                  const SizedBox(height: 24),
                  InkReveal(
                    whenVisible: true,
                    delay: const Duration(milliseconds: 80),
                    duration: const Duration(milliseconds: 650),
                    dy: 28,
                    child: peek,
                  ),
                ],
              ),
          ],
        ),
      ),
    );
  }
}
