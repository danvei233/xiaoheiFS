import 'simple_crud_screen.dart';

class PackagesScreen extends SimpleCrudScreen {
  PackagesScreen({super.key})
      : super(
          title: '套餐管理',
          listPath: '/admin/api/v1/packages',
          createPath: '/admin/api/v1/packages',
          updatePath: (item) => '/admin/api/v1/packages/${item['id']}',
          deletePath: (item) => '/admin/api/v1/packages/${item['id']}',
          fields: const [
            FieldDef(keyName: 'goods_type_id', label: '商品类型ID', type: FieldType.number, numberIsInt: true),
            FieldDef(keyName: 'plan_group_id', label: '套餐组ID', type: FieldType.number, numberIsInt: true),
            FieldDef(keyName: 'name', label: '名称', type: FieldType.text),
            FieldDef(keyName: 'cores', label: 'CPU核数', type: FieldType.number, numberIsInt: true),
            FieldDef(keyName: 'memory_gb', label: '内存(GB)', type: FieldType.number, numberIsInt: true),
            FieldDef(keyName: 'disk_gb', label: '硬盘(GB)', type: FieldType.number, numberIsInt: true),
            FieldDef(keyName: 'bandwidth_mbps', label: '带宽(Mbps)', type: FieldType.number, numberIsInt: true),
            FieldDef(keyName: 'monthly_price', label: '月费', type: FieldType.number),
            FieldDef(keyName: 'port_num', label: '端口数', type: FieldType.number, numberIsInt: true),
            FieldDef(keyName: 'active', label: '启用', type: FieldType.boolValue),
            FieldDef(keyName: 'visible', label: '可见', type: FieldType.boolValue),
            FieldDef(keyName: 'sort_order', label: '排序', type: FieldType.number, numberIsInt: true),
          ],
          titleBuilder: (item) => item['name']?.toString() ?? '',
          subtitleBuilder: (item) {
            final group = item['plan_group_name'] ?? item['plan_group_id'] ?? '';
            final type = item['goods_type_name'] ?? item['goods_type_id'] ?? '';
            final spec =
                '${item['cores'] ?? ''}C ${item['memory_gb'] ?? ''}G ${item['disk_gb'] ?? ''}G';
            final price = item['monthly_price']?.toString() ?? '';
            return '组 $group · 类型 $type · $spec · ￥$price';
          },
          lookupLoader: (client) async {
            final goodsResp = await client.getJson('/admin/api/v1/goods-types');
            final groupResp = await client.getJson('/admin/api/v1/plan-groups');
            final goodsItems = (goodsResp['items'] as List<dynamic>? ?? [])
                .map((e) => Map<String, dynamic>.from(e as Map))
                .toList();
            final groupItems = (groupResp['items'] as List<dynamic>? ?? [])
                .map((e) => Map<String, dynamic>.from(e as Map))
                .toList();
            final goodsMap = <int, String>{};
            final groupMap = <int, String>{};
            for (final item in goodsItems) {
              final id = item['id'] as int?;
              if (id == null) continue;
              goodsMap[id] = item['name']?.toString() ?? '';
            }
            for (final item in groupItems) {
              final id = item['id'] as int?;
              if (id == null) continue;
              groupMap[id] = item['name']?.toString() ?? '';
            }
            return {'goodsTypes': goodsMap, 'planGroups': groupMap};
          },
          enrichItem: (item, lookups) {
            final goodsTypes = lookups['goodsTypes'] as Map<int, String>? ?? {};
            final planGroups = lookups['planGroups'] as Map<int, String>? ?? {};
            final goodsId = item['goods_type_id'] as int?;
            final groupId = item['plan_group_id'] as int?;
            return {
              ...item,
              'goods_type_name': goodsId != null ? (goodsTypes[goodsId] ?? goodsId.toString()) : '',
              'plan_group_name': groupId != null ? (planGroups[groupId] ?? groupId.toString()) : '',
            };
          },
        );
}
