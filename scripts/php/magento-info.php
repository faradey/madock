<?php
/**
 * Magento 2 project facts for `madock info`.
 *
 *   php magento-info.php <site-root> [--format=text|json]
 *
 * Every non-Magento module in app/etc/config.php is matched to the composer
 * package that installed it by PATH, through vendor/composer/installed.json.
 * Matching by the `name` inside vendor/<pkg>/composer.json was the previous
 * approach and it silently dropped every package whose upstream had renamed
 * itself (the lock says one name, the vendored file another) — measured on a
 * live project: 136 modules in config.php, 135 printed, no message.
 *
 * Latest versions come from ONE `composer outdated --all` call for the whole
 * tree instead of one `composer show --all` per module: 135 network calls
 * became one, and the repository warnings composer prints on stderr for
 * every call no longer land between the lines of the report.
 */

$siteRootPath = rtrim($argv[1] ?? '', '/');
$format = 'text';
foreach (array_slice($argv, 2) as $arg) {
    if (strpos($arg, '--format=') === 0) {
        $format = substr($arg, strlen('--format='));
    }
}

$result = collect($siteRootPath);

if ($format === 'json') {
    echo json_encode($result, JSON_UNESCAPED_SLASHES | JSON_PRETTY_PRINT), "\n";
    exit(0);
}

printText($result);

/**
 * Collects the platform version and the third-party module table.
 *
 * Returns
 *   platform => [edition, constraint, installed]
 *   modules  => [[name, package, version, latest, enabled], ...]
 *   warnings => [string, ...]   what could not be established, and why
 */
function collect(string $root): array
{
    $out = [
        'platform' => ['edition' => 'Unknown', 'constraint' => null, 'installed' => null],
        'modules' => [],
        'warnings' => [],
    ];

    $configPath = $root . '/app/etc/config.php';
    if (!file_exists($configPath)) {
        $out['warnings'][] = 'app/etc/config.php not found — Magento is not installed here';
        return $out;
    }

    $composer = readJson($root . '/composer.json') ?? [];
    $installed = readInstalled($root);
    $psr4 = readPsr4($root);

    $require = $composer['require'] ?? [];
    $editions = [
        'magento/product-enterprise-edition' => 'Enterprise edition',
        'magento/product-community-edition' => 'Community edition',
        'magento/magento-cloud-metapackage' => 'Enterprise edition on Adobe Cloud',
    ];
    foreach ($editions as $package => $label) {
        if (!empty($require[$package])) {
            $out['platform']['edition'] = $label;
            $out['platform']['constraint'] = $require[$package];
            $out['platform']['installed'] = $installed['byName'][$package]['version'] ?? null;
            break;
        }
    }

    $latest = readLatest($root, $out['warnings']);

    $configData = include $configPath;
    foreach (($configData['modules'] ?? []) as $moduleName => $isActive) {
        if (strpos($moduleName, 'Magento_') === 0) {
            continue;
        }
        $row = [
            'name' => $moduleName,
            'package' => null,
            'version' => null,
            'latest' => null,
            'enabled' => (int) $isActive === 1,
        ];

        $namespace = str_replace('_', '\\', $moduleName) . '\\';
        $modulePath = $psr4[$namespace][0] ?? null;
        $package = $modulePath ? packageForPath($modulePath, $installed['byPath']) : null;

        if ($package !== null) {
            $row['package'] = $package['name'];
            $row['version'] = $package['version'];
            if ($latest !== null) {
                $row['latest'] = $latest[$package['name']] ?? null;
            }
        } else {
            // Not installed by composer: app/code, or a vendor path composer
            // does not know about. The module's own composer.json is the only
            // place a version can come from, and it is often absent.
            $composerJson = ($modulePath ?: appCodePath($root, $moduleName)) . '/composer.json';
            $data = readJson($composerJson);
            if ($data !== null) {
                $row['package'] = $data['name'] ?? null;
                $row['version'] = $data['version'] ?? null;
            }
        }

        $out['modules'][] = $row;
    }

    return $out;
}

function readJson(string $path): ?array
{
    if (!is_file($path)) {
        return null;
    }
    $data = json_decode((string) file_get_contents($path), true);
    return is_array($data) ? $data : null;
}

/**
 * vendor/composer/installed.json → two indexes: by package name and by the
 * real path of the install directory. Composer 2 wraps the list in
 * {"packages": [...]}; composer 1 wrote the bare list, and its entries carry
 * no install-path, so on composer 1 the path index stays empty.
 */
function readInstalled(string $root): array
{
    $byName = [];
    $byPath = [];
    $data = readJson($root . '/vendor/composer/installed.json') ?? [];
    $packages = $data['packages'] ?? (isset($data[0]) ? $data : []);
    foreach ($packages as $package) {
        if (empty($package['name'])) {
            continue;
        }
        $entry = [
            'name' => $package['name'],
            'version' => $package['version'] ?? null,
        ];
        $byName[$package['name']] = $entry;
        if (!empty($package['install-path'])) {
            $real = realpath($root . '/vendor/composer/' . $package['install-path']);
            if ($real !== false) {
                $byPath[$real] = $entry;
            }
        }
    }
    return ['byName' => $byName, 'byPath' => $byPath];
}

function readPsr4(string $root): array
{
    $path = $root . '/vendor/composer/autoload_psr4.php';
    if (!is_file($path)) {
        return [];
    }
    $map = include $path;
    return is_array($map) ? $map : [];
}

/**
 * Walks up from the module's autoload directory until it lands on a
 * directory composer installed a package into. A package that maps its
 * namespace to src/ is one level below its install path; a package holding
 * several modules maps them all to the same install path.
 */
function packageForPath(string $path, array $byPath): ?array
{
    $real = realpath($path);
    if ($real === false) {
        return null;
    }
    for ($i = 0; $i < 4; $i++) {
        if (isset($byPath[$real])) {
            return $byPath[$real];
        }
        $parent = dirname($real);
        if ($parent === $real) {
            break;
        }
        $real = $parent;
    }
    return null;
}

function appCodePath(string $root, string $moduleName): string
{
    return $root . '/app/code/' . str_replace('_', '/', $moduleName);
}

/**
 * One `composer outdated --all` for the whole tree. Returns name → latest, or
 * null when composer could not answer — and says so in $warnings rather than
 * quietly reporting the installed version as the latest, which is what the
 * previous script did.
 */
function readLatest(string $root, array &$warnings): ?array
{
    $cmd = 'composer outdated --all --format=json --no-interaction --no-ansi'
        . ' --working-dir=' . escapeshellarg($root) . ' 2>/dev/null';
    $lines = [];
    $code = 1;
    exec($cmd, $lines, $code);
    $data = json_decode(implode("\n", $lines), true);
    if ($code !== 0 || !is_array($data) || !isset($data['installed'])) {
        $warnings[] = 'latest versions unknown: `composer outdated --all` exited ' . $code;
        return null;
    }
    $latest = [];
    foreach ($data['installed'] as $package) {
        if (empty($package['name'])) {
            continue;
        }
        $version = $package['latest'] ?? null;
        // Composer writes "[none matched]" when the repository that holds the
        // package refused to answer — an expired subscription, mostly. That is
        // "unknown", not a version.
        if ($version === '[none matched]') {
            $version = null;
        }
        $latest[$package['name']] = $version;
    }
    return $latest;
}

function printText(array $result): void
{
    $p = $result['platform'];
    print("Magento Version\n");
    $line = $p['edition'];
    if ($p['constraint'] !== null) {
        $line .= ' ' . $p['constraint'];
    }
    if ($p['installed'] !== null) {
        $line .= ' (installed ' . $p['installed'] . ')';
    }
    print($line . "\n\n");

    print("Third-parties modules\n");
    print("Name, Package, Current version, Latest version, Status\n");
    foreach ($result['modules'] as $m) {
        print(implode(', ', [
            $m['name'],
            $m['package'] ?? '-',
            $m['version'] ?? '"no version"',
            $m['latest'] ?? '-',
            $m['enabled'] ? 'enabled' : 'disabled',
        ]) . "\n");
    }

    foreach ($result['warnings'] as $w) {
        print("\nWarning: " . $w . "\n");
    }
}
