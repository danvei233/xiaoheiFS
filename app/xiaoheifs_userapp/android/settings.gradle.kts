pluginManagement {
    val flutterSdkPath =
        run {
            val properties = java.util.Properties()
            file("local.properties").inputStream().use { properties.load(it) }
            val flutterSdkPath = properties.getProperty("flutter.sdk")
            require(flutterSdkPath != null) { "flutter.sdk not set in local.properties" }
            flutterSdkPath
        }

    includeBuild("$flutterSdkPath/packages/flutter_tools/gradle")

    repositories {\n        google()\n        mavenCentral()\n        gradlePluginPortal()\n        maven("https://maven.aliyun.com/repository/google")\n        maven("https://maven.aliyun.com/repository/central")\n        maven("https://maven.aliyun.com/repository/public")\n        maven("https://maven.aliyun.com/repository/gradle-plugin")\n    }
}

plugins {
    id("dev.flutter.flutter-plugin-loader") version "1.0.0"
    id("com.android.application") version "8.11.1" apply false
    id("org.jetbrains.kotlin.android") version "2.2.20" apply false
}

include(":app")

