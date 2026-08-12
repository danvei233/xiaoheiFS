package com.opensource.userapp

import android.net.Uri
import androidx.core.content.FileProvider
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import java.io.File

class MainActivity : FlutterActivity() {
    private val channelName = "xiaohei/update"

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, channelName)
            .setMethodCallHandler { call, result ->
                if (call.method == "getContentUriForPath") {
                    val path = call.argument<String>("path")
                    if (path.isNullOrEmpty()) {
                        result.error("bad_argument", "path required", null)
                        return@setMethodCallHandler
                    }
                    val file = File(path)
                    if (!file.exists()) {
                        result.error("not_found", "file does not exist", null)
                        return@setMethodCallHandler
                    }
                    // 只允许读取应用私有目录（cache/files）下的临时 APK，
                    // 防止把任意文件路径暴露成 content://。
                    val cacheDir = cacheDir.canonicalPath
                    val filesDir = filesDir.canonicalPath
                    val canonical = file.canonicalPath
                    if (!canonical.startsWith(cacheDir) && !canonical.startsWith(filesDir)) {
                        result.error("forbidden", "path outside app private dirs", null)
                        return@setMethodCallHandler
                    }
                    val uri: Uri = FileProvider.getUriForFile(
                        this, "$packageName.fileprovider", file
                    )
                    result.success(uri.toString())
                } else {
                    result.notImplemented()
                }
            }
    }
}
