import 'simple_crud_screen.dart';

class RegionsScreen extends SimpleCrudScreen {
  RegionsScreen({super.key})
      : super(
          title: '区域管理',
          listPath: '/admin/api/v1/regions',
          createPath: '/admin/api/v1/regions',
          updatePath: (item) => '/admin/api/v1/regions/${item['id']}',
          deletePath: (item) => '/admin/api/v1/regions/${item['id']}',
          fields: const [
            FieldDef(keyName: 'goods_type_id', label: '商品类型ID', type: FieldType.number, numberIsInt: true),
            FieldDef(keyName: 'code', label: '区域代码', type: FieldType.text),
            FieldDef(keyName: 'name', label: '区域名称', type: FieldType.text),
            FieldDef(keyName: 'active', label: '启用', type: FieldType.boolValue),
          ],
          titleBuilder: (item) => item['name']?.toString() ?? '',
          subtitleBuilder: (item) =>
              '代码 ${item['code'] ?? ''} · 类型 ${item['goods_type_name'] ?? item['goods_type_id'] ?? ''}',
          lookupLoader: (client) async {
            final resp = await client.getJson('/admin/api/v1/goods-types');
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
            return {'goodsTypes': map};
          },
          enrichItem: (item, lookups) {
            final goodsTypes = lookups['goodsTypes'] as Map<int, String>? ?? {};
            final id = item['goods_type_id'] as int?;
            return {
              ...item,
              'goods_type_name': id != null ? (goodsTypes[id] ?? id.toString()) : '',
            };
          },
        );
}
