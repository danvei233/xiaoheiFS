import 'simple_crud_screen.dart';

class LinesScreen extends SimpleCrudScreen {
  LinesScreen({super.key})
      : super(
          title: '线路管理',
          listPath: '/admin/api/v1/lines',
          createPath: '/admin/api/v1/lines',
          updatePath: (item) => '/admin/api/v1/lines/${item['id']}',
          deletePath: (item) => '/admin/api/v1/lines/${item['id']}',
          fields: const [
            FieldDef(keyName: 'region_id', label: '区域ID', type: FieldType.number, numberIsInt: true),
            FieldDef(keyName: 'name', label: '名称', type: FieldType.text),
            FieldDef(keyName: 'line_id', label: '线路ID', type: FieldType.number, numberIsInt: true),
            FieldDef(keyName: 'unit_core', label: 'CPU单价', type: FieldType.number),
            FieldDef(keyName: 'unit_mem', label: '内存单价', type: FieldType.number),
            FieldDef(keyName: 'unit_disk', label: '硬盘单价', type: FieldType.number),
            FieldDef(keyName: 'unit_bw', label: '带宽单价', type: FieldType.number),
            FieldDef(keyName: 'active', label: '启用', type: FieldType.boolValue),
            FieldDef(keyName: 'visible', label: '可见', type: FieldType.boolValue),
            FieldDef(keyName: 'sort_order', label: '排序', type: FieldType.number, numberIsInt: true),
          ],
          titleBuilder: (item) => item['name']?.toString() ?? '',
          subtitleBuilder: (item) =>
              '区域 ${item['region_name'] ?? item['region_id'] ?? ''} · 线路 ${item['line_id'] ?? ''}',
          lookupLoader: (client) async {
            final resp = await client.getJson('/admin/api/v1/regions');
            final items = (resp['items'] as List<dynamic>? ?? [])
                .map((e) => Map<String, dynamic>.from(e as Map))
                .toList();
            final map = <int, String>{};
            for (final item in items) {
              final id = item['id'] as int?;
              if (id == null) continue;
              final name = item['name']?.toString() ?? '';
              map[id] = name;
            }
            return {'regions': map};
          },
          enrichItem: (item, lookups) {
            final regions = lookups['regions'] as Map<int, String>? ?? {};
            final id = item['region_id'] as int?;
            return {
              ...item,
              'region_name': id != null ? (regions[id] ?? id.toString()) : '',
            };
          },
        );
}
