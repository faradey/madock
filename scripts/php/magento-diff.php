<?php
$siteRootPath = $argv[1] ?? '';
$oldArg = $argv[2] ?? '';
$newArg = $argv[3] ?? '';
$outputFile = $argv[4] ?? '';

if (empty($oldArg) || empty($newArg)) {
    fwrite(STDERR, "Usage: magento-diff.php <siteRoot> <oldVersion|oldPath> <newVersion|newPath> [outputFile]\n");
    fwrite(STDERR, "Examples:\n  magento-diff.php /var/www 2.4.8-p1 2.4.8-p2 var/diffs/magento.patch\n  magento-diff.php /var/www ../releases/2.4.6 ../releases/2.4.7\n");
    exit(1);
}

function normPath($siteRootPath, $path) {
    if (empty($path)) return '';
    // If absolute path use as is, else from site root
    if ($path[0] === '/') {
        return rtrim($path, "/\n\r");
    }
    return rtrim($siteRootPath . '/' . trim($path, '/'), "/\n\r");
}

function runCmdOrFail(string $cmd, int $exitCode = 10) {
    $output = [];
    $code = 0;
    exec($cmd, $output, $code);
    if ($code !== 0) {
        fwrite(STDERR, "Command failed ($code): $cmd\n" . implode("\n", $output) . "\n");
        exit($exitCode);
    }
}

$workBase = '/var/www/var/magento_diff';
$oldSrc = $workBase . '/src-old';
$newSrc = $workBase . '/src-new';

// Prepare work dirs
if (file_exists($oldSrc)) deleteDirectory($oldSrc);
if (file_exists($newSrc)) deleteDirectory($newSrc);
if (!file_exists($workBase)) mkdir($workBase, 0755, true);

$oldPath = null;
$newPath = null;

// If args are existing directories, use them; otherwise treat as Magento versions and download via composer
$oldAsPath = normPath($siteRootPath, $oldArg);
$newAsPath = normPath($siteRootPath, $newArg);

if (is_dir($oldAsPath) && is_dir($newAsPath)) {
    $oldPath = $oldAsPath;
    $newPath = $newAsPath;
} else {
    // Download Magento Open Source using composer create-project
    // Package: magento/project-community-edition
    $pkg = 'magento/project-community-edition';

    // Create-project syntax: composer create-project <package> <directory> <version>
    $composerCmdOld = 'cd ' . escapeshellarg($workBase) . ' && composer create-project --repository-url=https://repo.magento.com/ --no-interaction --no-plugins --ignore-platform-reqs ' . escapeshellarg($pkg) . ' ' . escapeshellarg($oldSrc) . ' ' . escapeshellarg($oldArg);
    $composerCmdNew = 'cd ' . escapeshellarg($workBase) . ' && composer create-project --repository-url=https://repo.magento.com/ --no-interaction --no-plugins --ignore-platform-reqs ' . escapeshellarg($pkg) . ' ' . escapeshellarg($newSrc) . ' ' . escapeshellarg($newArg);

    runCmdOrFail($composerCmdOld, 11);
    runCmdOrFail($composerCmdNew, 12);

    $oldPath = $oldSrc;
    $newPath = $newSrc;
}

$oldTmp = $workBase . '/old';
$newTmp = $workBase . '/new';

if (file_exists($oldTmp)) deleteDirectory($oldTmp);
if (file_exists($newTmp)) deleteDirectory($newTmp);
if (!file_exists($workBase)) mkdir($workBase, 0755, true);
mkdir($oldTmp, 0755, true);
mkdir($newTmp, 0755, true);

recurseCopy($oldPath, $oldTmp);
recurseCopy($newPath, $newTmp);

// Strip comments in both directories
stripCommentsInDir($oldTmp);
stripCommentsInDir($newTmp);

// Create diff
$cmd = 'diff -u -r -N ' . escapeshellarg($oldTmp) . ' ' . escapeshellarg($newTmp);
if (!empty($outputFile)) {
    $outputFile = normPath($siteRootPath, $outputFile);
    // Ensure output directory exists
    $outDir = dirname($outputFile);
    if (!file_exists($outDir)) mkdir($outDir, 0755, true);
    $cmd .= ' > ' . escapeshellarg($outputFile);
}

$output = null;
$responseCode = 0;
exec($cmd, $output, $responseCode);

if (empty($outputFile)) {
    // Print stdout output (when diff writes to stdout)
    if (is_array($output)) {
        echo implode("\n", $output);
    } else {
        // No differences or binary differences produce no stdout
        // Still echo nothing; rely on exit code
    }
} else {
    echo "Diff saved to: {$outputFile}\n";
}

exit($responseCode);

function stripCommentsInDir(string $dir)
{
    $rii = new RecursiveIteratorIterator(new RecursiveDirectoryIterator($dir, FilesystemIterator::SKIP_DOTS));
    $extMapPhp = ['php','phtml','inc'];
    $extMapJs = ['js','ts','tsx','jsx'];
    $extMapCss = ['css','less','scss','sass'];
    $extMapXml = ['xml'];
    $extMapHtml = ['html','htm'];
    $extMapOther = ['txt','md','svg']; // keep as is (but remove XML/HTML comments for svg)

    foreach ($rii as $file) {
        /** @var SplFileInfo $file */
        if (!$file->isFile()) continue;
        $path = $file->getPathname();
        // Ignore VCS dirs
        if (strpos($path, DIRECTORY_SEPARATOR.'.git'.DIRECTORY_SEPARATOR) !== false || strpos($path, DIRECTORY_SEPARATOR.'.idea'.DIRECTORY_SEPARATOR) !== false) {
            continue;
        }
        $ext = strtolower(pathinfo($path, PATHINFO_EXTENSION));
        $content = @file_get_contents($path);
        if ($content === false) continue;

        $original = $content;
        if (in_array($ext, $extMapPhp, true) || in_array($ext, $extMapJs, true) || in_array($ext, $extMapCss, true)) {
            // Remove /* ... */ including multiline
            $content = preg_replace('#/\*[^*]*\*+(?:[^/*][^*]*\*+)*/#s', '', $content);
            // Remove // ... to end of line
            $content = preg_replace('#(^|\s)//.*$#m', '$1', $content);
            // Remove # ... to end of line
            $content = preg_replace('/(^|\s)#.*$/m', '$1', $content);
        }
        if (in_array($ext, $extMapXml, true) || in_array($ext, $extMapHtml, true) || $ext === 'svg' || $ext === 'xhtml') {
            // Remove <!-- ... --> comments
            $content = preg_replace('#<!--([\s\S]*?)-->#', '', $content);
        }
        // Trim trailing spaces on each line to normalize minor diffs
        $content = preg_replace("#\s+$#m", '', $content);

        if ($content !== $original) {
            @file_put_contents($path, $content);
        }
    }
}

function deleteDirectory($dir) {
    if (!file_exists($dir)) {
        return true;
    }

    if (!is_dir($dir) || is_link($dir)) {
        return @unlink($dir);
    }

    foreach (scandir($dir) as $item) {
        if ($item == '.' || $item == '..') {
            continue;
        }

        if (!deleteDirectory($dir . "/" . $item)) {
            return false;
        }
    }

    return rmdir($dir);
}

function recurseCopy(
    string $sourceDirectory,
    string $destinationDirectory,
    string $childFolder = ''
) {
    $directory = opendir($sourceDirectory);

    if (is_dir($destinationDirectory) === false) {
        mkdir($destinationDirectory, 0755, true);
    }

    if ($childFolder !== '') {
        if (is_dir($destinationDirectory."/".$childFolder) === false) {
            mkdir($destinationDirectory."/".$childFolder, 0755, true);
        }

        while (($file = readdir($directory)) !== false) {
            if ($file === '.' || $file === '..') {
                continue;
            }

            if (is_dir($sourceDirectory."/".$file) === true) {
                recurseCopy($sourceDirectory."/".$file, $destinationDirectory."/".$childFolder/$file);
            } else {
                copy($sourceDirectory."/".$file, $destinationDirectory."/".$childFolder."/".$file);
            }
        }

        closedir($directory);

        return;
    }

    while (($file = readdir($directory)) !== false) {
        if ($file === '.' || $file === '..') {
            continue;
        }

        if (is_dir($sourceDirectory."/".$file) === true) {
            recurseCopy($sourceDirectory."/".$file, $destinationDirectory."/".$file);
        } else {
            copy($sourceDirectory."/".$file, $destinationDirectory."/".$file);
        }
    }

    closedir($directory);
}
