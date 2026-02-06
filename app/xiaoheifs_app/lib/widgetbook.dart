import 'package:flutter/material.dart';
import 'package:widgetbook/widgetbook.dart';

void main() {
  runApp(const AdminWidgetbook());
}

class AdminWidgetbook extends StatelessWidget {
  const AdminWidgetbook({super.key});

  @override
  Widget build(BuildContext context) {
    return Widgetbook.material(
      addons: const [
        InspectorAddon(),
      ],
      directories: [
        WidgetbookFolder(
          name: '基础',
          children: [
            WidgetbookComponent(
              name: 'Buttons',
              useCases: [
                WidgetbookUseCase(
                  name: 'Primary',
                  builder: (context) => Center(
                    child: FilledButton(
                      onPressed: () {},
                      child: const Text('Primary'),
                    ),
                  ),
                ),
              ],
            ),
          ],
        ),
      ],
    );
  }
}
