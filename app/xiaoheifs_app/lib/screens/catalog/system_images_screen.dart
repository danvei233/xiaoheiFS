import 'simple_crud_screen.dart';

class SystemImagesScreen extends SimpleCrudScreen {
  SystemImagesScreen({super.key})
      : super(
          title: '系统镜像',
          listPath: '/admin/api/v1/system-images',
          createPath: '/admin/api/v1/system-images',
          updatePath: (item) => '/admin/api/v1/system-images/${item['id']}',
          deletePath: (item) => '/admin/api/v1/system-images/${item['id']}',
          fields: const [
            FieldDef(keyName: 'image_id', label: '镜像ID', type: FieldType.number, numberIsInt: true),
            FieldDef(keyName: 'name', label: '名称', type: FieldType.text),
            FieldDef(keyName: 'type', label: '类型', type: FieldType.text),
            FieldDef(keyName: 'enabled', label: '启用', type: FieldType.boolValue),
          ],
          titleBuilder: (item) => item['name']?.toString() ?? '',
          subtitleBuilder: (item) {
            final type = _typeLabel(item['type']?.toString() ?? '');
            return 'ID ${item['image_id'] ?? ''} · 类型 $type';
          },
        );
}

String _typeLabel(String raw) {
  switch (raw.toLowerCase()) {
    case 'linux':
      return 'Linux';
    case 'windows':
      return 'Windows';
    default:
      return raw.isEmpty ? '-' : raw;
  }
}
