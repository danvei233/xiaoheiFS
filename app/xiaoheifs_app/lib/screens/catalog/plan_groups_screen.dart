import 'simple_crud_screen.dart';

class PlanGroupsScreen extends SimpleCrudScreen {
  PlanGroupsScreen({super.key})
      : super(
          title: '套餐组管理',
          listPath: '/admin/api/v1/plan-groups',
          createPath: '/admin/api/v1/plan-groups',
          updatePath: (item) => '/admin/api/v1/plan-groups/${item['id']}',
          deletePath: (item) => '/admin/api/v1/plan-groups/${item['id']}',
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
              '区域 ${item['region_name'] ?? item['region_id'] ?? ''} · 线路 ${item['line_name'] ?? item['line_id'] ?? ''}',
          lookupLoader: (client) async {
            final regionsResp = await client.getJson('/admin/api/v1/regions');
            final linesResp = await client.getJson('/admin/api/v1/lines');
            final regions = (regionsResp['items'] as List<dynamic>? ?? [])
                .map((e) => Map<String, dynamic>.from(e as Map))
                .toList();
            final lines = (linesResp['items'] as List<dynamic>? ?? [])
                .map((e) => Map<String, dynamic>.from(e as Map))
                .toList();
            final regionMap = <int, String>{};
            final lineMap = <int, String>{};
            for (final item in regions) {
              final id = item['id'] as int?;
              if (id == null) continue;
              regionMap[id] = item['name']?.toString() ?? '';
            }
            for (final item in lines) {
              final id = item['id'] as int?;
              if (id == null) continue;
              lineMap[id] = item['name']?.toString() ?? '';
            }
            return {'regions': regionMap, 'lines': lineMap};
          },
          enrichItem: (item, lookups) {
            final regions = lookups['regions'] as Map<int, String>? ?? {};
            final lines = lookups['lines'] as Map<int, String>? ?? {};
            final regionId = item['region_id'] as int?;
            final lineId = item['line_id'] as int?;
            return {
              ...item,
              'region_name': regionId != null ? (regions[regionId] ?? regionId.toString()) : '',
              'line_name': lineId != null ? (lines[lineId] ?? lineId.toString()) : '',
            };
          },
        );
}
