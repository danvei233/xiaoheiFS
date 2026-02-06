import 'simple_crud_screen.dart';

class BillingCyclesScreen extends SimpleCrudScreen {
  BillingCyclesScreen({super.key})
      : super(
          title: '计费周期',
          listPath: '/admin/api/v1/billing-cycles',
          createPath: '/admin/api/v1/billing-cycles',
          updatePath: (item) => '/admin/api/v1/billing-cycles/${item['id']}',
          deletePath: (item) => '/admin/api/v1/billing-cycles/${item['id']}',
          fields: const [
            FieldDef(keyName: 'name', label: '名称', type: FieldType.text),
            FieldDef(keyName: 'months', label: '月数', type: FieldType.number, numberIsInt: true),
            FieldDef(keyName: 'multiplier', label: '倍率', type: FieldType.number),
            FieldDef(keyName: 'min_qty', label: '最小数量', type: FieldType.number, numberIsInt: true),
            FieldDef(keyName: 'max_qty', label: '最大数量', type: FieldType.number, numberIsInt: true),
            FieldDef(keyName: 'active', label: '启用', type: FieldType.boolValue),
            FieldDef(keyName: 'sort_order', label: '排序', type: FieldType.number, numberIsInt: true),
          ],
          titleBuilder: (item) => item['name']?.toString() ?? '',
          subtitleBuilder: (item) =>
              '${item['months'] ?? ''} 月 · 倍率 ${item['multiplier'] ?? ''} · 数量 ${item['min_qty'] ?? ''}-${item['max_qty'] ?? ''}',
        );
}
