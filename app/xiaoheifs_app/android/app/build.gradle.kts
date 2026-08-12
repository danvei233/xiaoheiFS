buildscript {
    dependencies {
        // google-services disabled for open-source build
    }
}

plugins {
    id("com.android.application")
    id("kotlin-android")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

android {
    namespace = "com.opensource.admin"
    compileSdk = flutter.compileSdkVersion
    ndkVersion = flutter.ndkVersion

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
        isCoreLibraryDesugaringEnabled = true
    }

    kotlinOptions {
        jvmTarget = JavaVersion.VERSION_17.toString()
    }

    defaultConfig {
        applicationId = "com.opensource.admin"
        minSdk = flutter.minSdkVersion
        targetSdk = flutter.targetSdkVersion
        versionCode = flutter.versionCode
        versionName = flutter.versionName

        // 生产构建默认禁止明文 HTTP；debug 构建放开以支持本地开发联调。
        manifestPlaceholders["usesCleartextTraffic"] = "false"
    }

    signingConfigs {
        create("release") {
            // Release 签名完全来自 CI/构建环境注入，绝不回退到 debug keystore。
            val keystorePath = System.getenv("ANDROID_RELEASE_KEYSTORE_PATH")
            if (!keystorePath.isNullOrBlank()) {
                storeFile = file(keystorePath)
                storePassword = System.getenv("ANDROID_RELEASE_KEYSTORE_PASSWORD") ?: ""
                keyAlias = System.getenv("ANDROID_RELEASE_KEY_ALIAS") ?: ""
                keyPassword = System.getenv("ANDROID_RELEASE_KEY_PASSWORD") ?: ""
            }
        }
    }

    buildTypes {
        debug {
            manifestPlaceholders["usesCleartextTraffic"] = "true"
        }
        release {
            manifestPlaceholders["usesCleartextTraffic"] = "false"
            val keystorePath = System.getenv("ANDROID_RELEASE_KEYSTORE_PATH")
            if (!keystorePath.isNullOrBlank()) {
                signingConfig = signingConfigs.getByName("release")
            }
            // 未注入 keystore 时保持未签名（unsigned APK 无法安装/上架），
            // 拒绝静默使用 debug 签名。CI 在 release 构建前强制校验该环境变量。
        }
    }
}

flutter {
    source = "../.."
}

dependencies {
    coreLibraryDesugaring("com.android.tools:desugar_jdk_libs:2.1.4")
}
// google-services plugin disabled for open-source build
