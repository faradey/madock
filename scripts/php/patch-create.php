<?php
$patchContainerPath = "/var/www/patches";
$siteRootPath = $argv[1];

$vendorPath = $siteRootPath."/vendor";
$patchMagentoPath = $siteRootPath."/patches/composer";
$filePatch = $siteRootPath."/".trim($argv[2],"/");
$patchName = $argv[3]??"";
$patchTitle = $argv[4]?:$patchName;
$force = $argv[5]??"";

function detectComposerPatchesMajor(string $siteRootPath): int {
    $lockPath = $siteRootPath."/composer.lock";
    if(!file_exists($lockPath)){
        return 1;
    }
    $lock = json_decode(file_get_contents($lockPath), true);
    if(!is_array($lock)){
        return 1;
    }
    foreach(['packages', 'packages-dev'] as $section){
        if(empty($lock[$section]) || !is_array($lock[$section])){
            continue;
        }
        foreach($lock[$section] as $pkg){
            if(($pkg['name'] ?? '') !== 'cweagans/composer-patches'){
                continue;
            }
            $version = ltrim((string)($pkg['version'] ?? ''), 'v');
            $parts = explode('.', $version);
            if(isset($parts[0]) && is_numeric($parts[0])){
                return (int)$parts[0];
            }
        }
    }
    return 1;
}

function invalidatePatchesLock(string $siteRootPath, int $major): void {
    if($major < 2){
        return;
    }
    $lockPath = $siteRootPath."/patches.lock.json";
    if(file_exists($lockPath)){
        @unlink($lockPath);
        print("patches.lock.json removed (cweagans >=2). Run `composer install` or `composer patches-relock` to regenerate.\n");
    }
}

function setPatchEntry(array &$patches, string $module, string $title, string $path, int $major, bool $force): bool {
    if($major >= 2){
        if(!isset($patches[$module]) || !is_array($patches[$module])){
            $patches[$module] = [];
        }
        $normalized = [];
        foreach($patches[$module] as $key => $existing){
            if(is_array($existing)){
                $normalized[] = [
                    'description' => (string)($existing['description'] ?? (is_string($key) ? $key : '')),
                    'url'         => (string)($existing['url'] ?? ''),
                ];
            } elseif(is_string($existing)){
                $normalized[] = [
                    'description' => is_string($key) ? $key : '',
                    'url'         => $existing,
                ];
            }
        }
        $patches[$module] = $normalized;
        $duplicate = false;
        foreach($patches[$module] as $idx => $existing){
            if(($existing['description'] ?? '') === $title || ($existing['url'] ?? '') === $path){
                if(!$force){
                    return false;
                }
                unset($patches[$module][$idx]);
                $duplicate = true;
            }
        }
        if($duplicate){
            $patches[$module] = array_values($patches[$module]);
        }
        $patches[$module][] = [
            'description' => $title,
            'url'         => $path,
        ];
        return true;
    }
    if(!$force && !empty($patches[$module][$title])){
        return false;
    }
    if(!isset($patches[$module]) || !is_array($patches[$module])){
        $patches[$module] = [];
    }
    $patches[$module][$title] = $path;
    return true;
}

if(file_exists($filePatch)){
    $composerPatchesMajor = detectComposerPatchesMajor($siteRootPath);
    $moduleRoot = explode("vendor", $filePatch, 2)[1]??null;
    if($moduleRoot){
        $moduleRoot = explode("/", trim($moduleRoot, "/"), 3);
        $moduleComposerPath = $vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]."/composer.json";
        if(empty($patchName)) {
            $patchName = $moduleRoot[0]."_".$moduleRoot[1].".patch";
        }
        if(empty($patchTitle)) {
            $patchTitle = "Fix ".$moduleRoot[0]."/".$moduleRoot[1]." module";
        }
        if(file_exists($moduleComposerPath)){
            $composerJsonData = file_get_contents($moduleComposerPath);
            $jsonData = json_decode($composerJsonData, true);
            $composerModuleName = $jsonData['name'];
            $composerModuleNameDir = str_replace("/", "_", str_replace("//", "//", $jsonData['name']));
            $composerModuleVersion = $jsonData['version'];

            try {
                if(file_exists($patchContainerPath."/".$composerModuleNameDir)){
                    deleteDirectory($patchContainerPath."/".$composerModuleNameDir);
                }
                mkdir($patchContainerPath."/".$composerModuleNameDir);
                recurseCopy($vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1], $patchContainerPath."/".$composerModuleNameDir);
                deleteDirectory($vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]);
            
                $output = null;
                $responseCode = 0;
                exec("cd ".$siteRootPath." && composer install --no-plugins --no-scripts --ignore-platform-reqs", $output, $responseCode);
                if($responseCode != 0){
                    recurseCopy($patchContainerPath."/".$composerModuleNameDir, $vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]);
                    if(file_exists($patchContainerPath."/".$composerModuleNameDir)){
                        deleteDirectory($patchContainerPath."/".$composerModuleNameDir);
                    }
                    if(file_exists($patchContainerPath."/vendor")){
                        deleteDirectory($patchContainerPath."/vendor");
                    }
                    mkdir($patchContainerPath."/vendor");
                    recurseCopy($vendorPath, $patchContainerPath."/vendor");
                    deleteDirectory($vendorPath);

                    $output = null;
                    $responseCode = 0;
                    exec("cd ".$siteRootPath." && composer install --no-plugins --no-scripts --ignore-platform-reqs", $output, $responseCode);
                    
                    if($responseCode != 0){
                        exec("rm -r ".$vendorPath."/" . $moduleRoot[0]."/".$moduleRoot[1]);
                        recurseCopy($patchContainerPath."/vendor", $vendorPath);
                        print_r($output);
                    }
                } else {
                    $composerFile = $siteRootPath."/composer.json";
                    $composerJsonData = json_decode(file_get_contents($composerFile), true);
                    if(!empty($composerJsonData['extra']['patches-search'])) {
                        $patchSearchFolder = $composerJsonData['extra']['patches-search'];
                        if(is_array($patchSearchFolder)){
                            $patchSearchFolder = $patchSearchFolder[0];
                        }
                        $patchMagentoPath = $siteRootPath."/".trim($patchSearchFolder, "/");
                    }
                    if(!empty($force) || !file_exists($patchMagentoPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]."/".$patchName)){
                        if(!file_exists($patchMagentoPath . "/" . $moduleRoot[0] . "/" . $moduleRoot[1])){
                            mkdir($patchMagentoPath . "/" . $moduleRoot[0] . "/" . $moduleRoot[1], 0755, true);
                        }
                        if(empty($moduleRoot[2]) && is_dir($vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1])){
                            exec("diff -u -r -N ".$vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]. " ".$patchContainerPath."/".$composerModuleNameDir . " > " .$patchMagentoPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]."/".$patchName, $output, $responseCode);
                            print("diff -u -r -N ".$vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]. " ".$patchContainerPath."/".$composerModuleNameDir . " > " .$patchMagentoPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]."/".$patchName);
                        } else {
                            $moduleRoot[2] = trim($moduleRoot[2], "/");
                            exec("diff -u ".$vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]."/".$moduleRoot[2]. " ".$patchContainerPath."/".$composerModuleNameDir."/".$moduleRoot[2] . " > ".$patchMagentoPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]."/".$patchName, $output, $responseCode);
                        }

                        $patchContent = file_get_contents($patchMagentoPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]."/".$patchName);
                        $patchContent = str_replace($vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1], "", $patchContent);
                        $patchContent = str_replace($patchContainerPath."/".$composerModuleNameDir, "", $patchContent);
                        $patchContent = preg_replace_callback(
                            '/^(---|\+\+\+)[ \t]+(\S+)(?:[ \t]+[^\n]*)?$/m',
                            function($m){
                                $prefix = $m[1] === '---' ? 'a' : 'b';
                                $path = $m[2];
                                if($path === '/dev/null' || $path === 'dev/null'){
                                    return $m[1].' /dev/null';
                                }
                                if(preg_match('#^[ab]/#', $path)){
                                    return $m[1].' '.$path;
                                }
                                $rel = ltrim($path, '/');
                                if($rel === ''){
                                    return $m[1].' /dev/null';
                                }
                                return $m[1].' '.$prefix.'/'.$rel;
                            },
                            $patchContent
                        );
                        if(!empty($composerJsonData['extra']['patches-search'])) {
                            $patchContent = "@package ".$moduleRoot[0]."/".$moduleRoot[1]."\n\n".$patchContent;
                        }
                        file_put_contents($patchMagentoPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]."/".$patchName, $patchContent);

                        $module = $moduleRoot[0]."/".$moduleRoot[1];
                        $relPath = "patches/composer/".$moduleRoot[0]."/".$moduleRoot[1]."/".$patchName;
                        if(isset($composerJsonData['extra']['patches'])) {
                            if(!is_array($composerJsonData['extra']['patches'])){
                                $composerJsonData['extra']['patches'] = [];
                            }
                            if(setPatchEntry($composerJsonData['extra']['patches'], $module, $patchTitle, $relPath, $composerPatchesMajor, !empty($force))){
                                file_put_contents($composerFile, json_encode($composerJsonData, JSON_PRETTY_PRINT|JSON_UNESCAPED_SLASHES));
                                invalidatePatchesLock($siteRootPath, $composerPatchesMajor);
                                print("\nThe patch was created successfully\n");
                            } else {
                                print("The patch with same title or name has already been created.\n");
                            }
                        } elseif(isset($composerJsonData['extra']['patches-file']) || file_exists($siteRootPath."/patches.json")) {
                            $patchesFile = $composerJsonData['extra']['patches-file']??'patches.json';
                            if(is_array($patchesFile)){
                                $patchesFile = $patchesFile[0];
                            }
                            $composerPatchesFile = $siteRootPath."/".$patchesFile;
                            $composerPatchesJsonData = json_decode(file_get_contents($composerPatchesFile), true);
                            if(!is_array($composerPatchesJsonData)){
                                $composerPatchesJsonData = [];
                            }
                            if(!isset($composerPatchesJsonData['patches']) || !is_array($composerPatchesJsonData['patches'])){
                                $composerPatchesJsonData['patches'] = [];
                            }
                            if(setPatchEntry($composerPatchesJsonData['patches'], $module, $patchTitle, $relPath, $composerPatchesMajor, !empty($force))){
                                file_put_contents($composerPatchesFile, json_encode($composerPatchesJsonData, JSON_PRETTY_PRINT|JSON_UNESCAPED_SLASHES));
                                invalidatePatchesLock($siteRootPath, $composerPatchesMajor);
                                print("\nThe patch was created successfully\n");
                            } else {
                                print("The patch with same title or name has already been created.\n");
                            }
                        } elseif(empty($composerJsonData['extra']['patches-search'])) {
                            if(!isset($composerJsonData['extra'])) {
                                $composerJsonData['extra'] = [];
                            }
                            $composerJsonData['extra']['patches'] = [];
                            setPatchEntry($composerJsonData['extra']['patches'], $module, $patchTitle, $relPath, $composerPatchesMajor, true);
                            file_put_contents($composerFile, json_encode($composerJsonData, JSON_PRETTY_PRINT|JSON_UNESCAPED_SLASHES));
                            invalidatePatchesLock($siteRootPath, $composerPatchesMajor);
                            print("\nThe patch was created successfully\n");
                        }
                        exec("rm -r ".$vendorPath."/" . $moduleRoot[0]."/".$moduleRoot[1]);
                        recurseCopy($patchContainerPath."/".$composerModuleNameDir, $vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]);
                    } else {
                        print("The patch with same title or name is already exists.\n");
                    }
                }
            } catch(\Exception | \Error $e) {
                print($e->getMessage()."\n");
            }
        }
    }
} else {
    print($filePatch." is not exist.\n");
}

function deleteDirectory($dir) {
    if (!file_exists($dir)) {
        return true;
    }

    if (!is_dir($dir) || is_link($dir)) {
        return unlink($dir);
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
    string $destinationDirectory
) {
    $directory = opendir($sourceDirectory);

    if (is_dir($destinationDirectory) === false) {
        mkdir($destinationDirectory, 0755, true);
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
