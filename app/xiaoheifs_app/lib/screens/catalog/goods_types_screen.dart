import 'simple_crud_screen.dart';

class GoodsTypesScreen extends SimpleCrudScreen {
  GoodsTypesScreen({super.key})
      : super(
          title: '商品类型',
          listPath: '/admin/api/v1/goods-types',
          createPath: '/admin/api/v1/goods-types',
          updatePath: (item) => '/admin/api/v1/goods-types/${item['id']}',
          deletePath: (item) => '/admin/api/v1/goods-types/${item['id']}',
          fields: const [
            FieldDef(keyName: 'code', label: '代码', type: FieldType.text),
            FieldDef(keyName: 'name', label: '名称', type: FieldType.text),
            FieldDef(keyName: 'active', label: '启用', type: FieldType.boolValue),
            FieldDef(keyName: 'sort_order', label: '排序', type: FieldType.number, numberIsInt: true),
            FieldDef(keyName: 'automation_plugin_id', label: '插件ID', type: FieldType.text),
            FieldDef(keyName: 'automation_instance_id', label: '插件实例', type: FieldType.text),
          ],
          titleBuilder: (item) => item['name']?.toString() ?? '',
          subtitleBuilder: (item) {
            final code = item['code'] ?? '';
            final plugin = item['automation_plugin_id'] ?? '-';
            final instance = item['automation_instance_id'] ?? '-';
            return '代码 $code · 插件 $plugin / $instance';
          },
        );
}
