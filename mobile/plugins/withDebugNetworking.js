// Native build settings. In debug builds only:
//  1. Cleartext to the emulator's host (10.0.2.2) and localhost, so a debug APK can
//     talk to a local API and Metro. Everything lives under src/debug, so release
//     builds never see it and stay HTTPS-only.
//  2. The JS bundle is embedded in debug APKs too, so one runs without Metro. If
//     Metro is running, React Native still prefers it.
const fs = require('fs');
const path = require('path');
const { withAppBuildGradle, withDangerousMod } = require('expo/config-plugins');

const NETWORK_CONFIG = `<?xml version="1.0" encoding="utf-8"?>
<network-security-config>
  <base-config cleartextTrafficPermitted="false" />
  <domain-config cleartextTrafficPermitted="true">
    <domain includeSubdomains="false">10.0.2.2</domain>
    <domain includeSubdomains="false">localhost</domain>
    <domain includeSubdomains="false">127.0.0.1</domain>
  </domain-config>
</network-security-config>
`;

function withDebugCleartext(config) {
  return withDangerousMod(config, [
    'android',
    (cfg) => {
      const debugDir = path.join(cfg.modRequest.platformProjectRoot, 'app', 'src', 'debug');
      const xmlDir = path.join(debugDir, 'res', 'xml');
      fs.mkdirSync(xmlDir, { recursive: true });
      fs.writeFileSync(path.join(xmlDir, 'debug_network_security_config.xml'), NETWORK_CONFIG);

      const manifestPath = path.join(debugDir, 'AndroidManifest.xml');
      const manifest = fs.existsSync(manifestPath)
        ? fs.readFileSync(manifestPath, 'utf8')
        : '<?xml version="1.0" encoding="utf-8"?>\n<manifest xmlns:android="http://schemas.android.com/apk/res/android" xmlns:tools="http://schemas.android.com/tools">\n  <application tools:targetApi="28"/>\n</manifest>\n';
      if (!manifest.includes('networkSecurityConfig')) {
        fs.writeFileSync(
          manifestPath,
          manifest.replace('<application', '<application android:networkSecurityConfig="@xml/debug_network_security_config"'),
        );
      }
      return cfg;
    },
  ]);
}

function withEmbeddedDebugBundle(config) {
  return withAppBuildGradle(config, (cfg) => {
    const marker = 'debuggableVariants = []';
    if (!cfg.modResults.contents.includes(marker)) {
      cfg.modResults.contents = cfg.modResults.contents.replace(
        /react\s*\{/,
        (m) => `${m}\n    // Embed the JS bundle in debug APKs too (see plugins/withDebugNetworking.js).\n    ${marker}\n`,
      );
    }
    return cfg;
  });
}

// Every build, on every OS: compile each native library's generated code as one unity file.
// CMake otherwise names objects after their absolute source path, which embeds the project
// path a second time and passes Windows' 260-character limit even from a short drive.
function withUnityNativeBuild(config) {
  return withAppBuildGradle(config, (cfg) => {
    const marker = '-DCMAKE_UNITY_BUILD=ON';
    if (!cfg.modResults.contents.includes(marker)) {
      cfg.modResults.contents += `\nandroid { defaultConfig { externalNativeBuild { cmake { arguments "${marker}" } } } }\n`;
    }
    return cfg;
  });
}

module.exports = (config) => withUnityNativeBuild(withEmbeddedDebugBundle(withDebugCleartext(config)));
