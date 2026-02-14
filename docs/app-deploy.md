# App 部署教程

本文档描述 `app/` 下 Flutter 项目的构建、部署与常见问题处理。

## 1. 适用项目
- `app/xiaoheifs_app`：管理端 App
- `app/xiaoheifs_userapp`：用户端 App

## 2. 前置条件
- Flutter SDK（建议与项目当前锁定版本一致）
- Dart SDK（随 Flutter）
- JDK 17+
- Android SDK / Android Studio（Android 打包）
- Visual Studio Build Tools（Windows 打包）

可执行检查：
```bash
flutter --version
flutter doctor -v
```

## 3. 部署前配置（开源版必看）
- 不要携带生产 `api key`、`token`、私有域名、私有更新源。
- 包名、默认 API 地址、更新地址请替换为公开安全值。
- Firebase 若未配置完整，请禁用对应插件和初始化逻辑，避免构建失败。

## 4. Android 构建

### 4.1 管理端 App
```bash
cd app/xiaoheifs_app
flutter pub get
flutter build apk --release
```
产物：`build/app/outputs/flutter-apk/app-release.apk`

### 4.2 用户端 App
```bash
cd app/xiaoheifs_userapp
flutter pub get
flutter build apk --release
```
产物：`build/app/outputs/flutter-apk/app-release.apk`

### 4.3 验证
- APK 生成成功
- 安装后能启动到登录页
- 默认接口地址正确

## 5. Windows 构建

### 5.1 管理端 App
```powershell
cd app\xiaoheifs_app
flutter pub get
flutter build windows --release
```

### 5.2 用户端 App
```powershell
cd app\xiaoheifs_userapp
flutter pub get
flutter build windows --release
```

产物目录通常在：
- `build/windows/x64/runner/Release/`

## 6. 发布建议
- 为每次发布打 tag 并记录 commit hash。
- 提供 SHA256 校验文件。
- 在发布说明中标注包名、接口地址与版本号。

## 7. 常见问题排查

### 7.1 `google-services` / Firebase 相关构建错误
- 检查是否误启用 `com.google.gms.google-services`。
- 检查是否仍引用 `Firebase.initializeApp()`。
- 开源版无 Firebase 配置时，建议完全禁用对应代码路径。

### 7.2 Windows/Android 路径问题
- 如果项目路径含中文导致 Android 构建报错，可在 `android/gradle.properties` 增加：
```properties
android.overridePathCheck=true
```

### 7.3 依赖下载慢或失败
- 切换网络环境后重试。
- 执行 `flutter clean` 后再 `flutter pub get`。

## 8. 验收清单
- 能构建出产物（APK/Windows）
- 首屏可正常显示
- 登录/退出正常
- 关键页面无明显崩溃
- 默认配置不包含隐私数据
