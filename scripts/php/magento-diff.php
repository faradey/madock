<?php
$siteRootPath = $argv[1] ?? '';
$oldArg = $argv[2] ?? '';
$newArg = $argv[3] ?? '';
$publicRel = $argv[4] ?? '';

if (empty($oldArg) || empty($newArg)) {
    fwrite(STDERR, "Usage: magento-diff.php <siteRoot> <oldVersion|oldPath> <newVersion|newPath> [<publicDirFromSiteRoot>]\n");
    fwrite(STDERR, "Notes: Generates per-file diffs into /var/www/var/magento_diff/diffs without console output. Public output defaults to /var/www/html/diffs but can be overridden by the 4th argument (path from site root).\n");
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

function slugifyArg(string $s): string {
    $s = trim($s);
    // If it's a path, reduce to a representative form
    // Replace directory separators with dashes
    $s = str_replace(['\\\\/','\\'], '/', $s);
    $s = str_replace('/', '-', $s);
    // Keep only safe chars: letters, digits, dot, dash, underscore
    $s = preg_replace('/[^A-Za-z0-9._-]+/', '-', $s);
    $s = trim($s, '-._');
    if ($s === '') {
        $s = 'unknown';
    }
    return $s;
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

// Create per-file diffs for specific file types only
$outBase = $workBase . '/diffs';
if (file_exists($outBase)) deleteDirectory($outBase);
mkdir($outBase, 0755, true);

$allowedExts = ['php','phtml','html','xml','js','css','less','sass','yaml'];
$allowedNames = ['composer.json','php.ini'];

$oldList = listAllowedFiles($oldTmp, $allowedExts, $allowedNames);
$newList = listAllowedFiles($newTmp, $allowedExts, $allowedNames);
$allRel = array_unique(array_merge(array_keys($oldList), array_keys($newList)));
sort($allRel);

$diffsCount = 0;
foreach ($allRel as $rel) {
    $oldFile = $oldTmp . '/' . $rel;
    $newFile = $newTmp . '/' . $rel;
    $outFile = $outBase . '/' . $rel . '.patch';
    $outDir = dirname($outFile);

    $oldArgFile = file_exists($oldFile) ? escapeshellarg($oldFile) : '/dev/null';
    $newArgFile = file_exists($newFile) ? escapeshellarg($newFile) : '/dev/null';

    // Build labels for diff headers to include relative paths
    $labelOld = file_exists($oldFile) ? escapeshellarg('a/' . $rel) : escapeshellarg('a/DEV_NULL');
    $labelNew = file_exists($newFile) ? escapeshellarg('b/' . $rel) : escapeshellarg('b/DEV_NULL');

    $cmd = 'diff -u -N --label ' . $labelOld . ' --label ' . $labelNew . ' ' . $oldArgFile . ' ' . $newArgFile;
    $output = [];
    $code = 0;
    exec($cmd, $output, $code);
    if ($code === 1 && !empty($output)) { // differences found
        if (!file_exists($outDir)) mkdir($outDir, 0755, true);
        file_put_contents($outFile, implode("\n", $output) . "\n");
        $diffsCount++;
    } else {
        // No differences; ensure no leftover empty file
        if (file_exists($outFile)) @unlink($outFile);
    }
}

// After creating diffs, move them to <publicBase>/magento-{old}-{new}
$publicBase = !empty($publicRel) ? normPath($siteRootPath, $publicRel) : '/var/www/html/diffs';
$oldSlug = slugifyArg($oldArg);
$newSlug = slugifyArg($newArg);
$targetDir = $publicBase . '/magento-' . $oldSlug . '-' . $newSlug;

if (!file_exists($publicBase)) {
    @mkdir($publicBase, 0755, true);
}
if (file_exists($targetDir)) {
    deleteDirectory($targetDir);
}
@mkdir($targetDir, 0755, true);
recurseCopy($outBase, $targetDir);
// Remove any empty directories in the target directory after copy
removeEmptyDirectories($targetDir);
// Optionally clean up working diffs directory
deleteDirectory($outBase);

// Do not print to console; exit 0 regardless
if ($diffsCount > 0) {
    fwrite(STDERR, "Generated $diffsCount diffs in $targetDir\n");
} else {
    fwrite(STDERR, "No differences found between the specified Magento versions.\n");
}

exit(0);

function listAllowedFiles(string $root, array $allowedExts, array $allowedNames): array
{
    $root = rtrim($root, '/');
    $rii = new RecursiveIteratorIterator(new RecursiveDirectoryIterator($root, FilesystemIterator::SKIP_DOTS));
    $result = [];
    foreach ($rii as $file) {
        /** @var SplFileInfo $file */
        if (!$file->isFile()) continue;
        $path = $file->getPathname();
        if (strpos($path, DIRECTORY_SEPARATOR.'.git'.DIRECTORY_SEPARATOR) !== false || strpos($path, DIRECTORY_SEPARATOR.'.idea'.DIRECTORY_SEPARATOR) !== false) {
            continue;
        }
        $rel = ltrim(str_replace($root.'/', '', $path), '/');
        $name = basename($path);
        $ext = strtolower(pathinfo($path, PATHINFO_EXTENSION));
        if (in_array($name, $allowedNames, true) || in_array($ext, $allowedExts, true)) {
            $result[$rel] = true;
        }
    }
    return $result;
}

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
            //$content = preg_replace('#(^|\s)//.*$#m', '$1', $content);
            // Remove # ... to end of line
            //$content = preg_replace('/(^|\s)#.*$/m', '$1', $content);
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

/**
 * Remove empty directories within the given directory (does not remove the root itself).
 */
function removeEmptyDirectories(string $dir): void {
    if (!is_dir($dir)) {
        return;
    }
    // Depth-first removal
    $items = @scandir($dir);
    if ($items === false) return;
    foreach ($items as $item) {
        if ($item === '.' || $item === '..') continue;
        $path = $dir . DIRECTORY_SEPARATOR . $item;
        if (is_dir($path)) {
            removeEmptyDirectories($path);
            // After pruning children, if directory is empty, remove it
            $subItems = @scandir($path);
            if ($subItems !== false) {
                $count = 0;
                foreach ($subItems as $si) {
                    if ($si !== '.' && $si !== '..') {
                        $count++;
                        break;
                    }
                }
                if ($count === 0) {
                    @rmdir($path);
                }
            }
        }
    }
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
                recurseCopy($sourceDirectory."/".$file, $destinationDirectory."/".$childFolder."/".$file);
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
