<?php
$patchContainerPath = "/var/www/patches";
$siteRootPath = "/var/www/html";

$vendorPath = $siteRootPath."/vendor";
$patchMagentoPath = $siteRootPath."/patches/composer";
$filePatch = $siteRootPath."/".trim($argv[1],"/");
$patchName = $argv[2];

if(file_exists($filePatch)){
    $moduleRoot = explode("vendor", $filePatch, 2)[1]??null;
    if($moduleRoot){
        $moduleRoot = explode("/", trim($moduleRoot, "/"), 3);
        $moduleComposerPath = $vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]."/composer.json";
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

                exec("cd ".$siteRootPath." && composer install", $output, $responseCode);
                if($responseCode != 0){
                    print_r($output);
                } else {
                    $moduleRoot[2] = trim($moduleRoot[2], "/");
                    copy($patchContainerPath."/".$composerModuleNameDir."/".$moduleRoot[2], $vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]."/".$moduleRoot[2].".new");
                    exec("cd ".$vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]." && diff -u ".$moduleRoot[2]. " ".$moduleRoot[2] . ".new > ".$patchName, $output, $responseCode);
                    
                    if(file_exists($vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]."/".$patchName)){
                        $patchContent = file_get_contents($vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]."/".$patchName);
                        $patchContent = str_replace($moduleRoot[2].".new", $moduleRoot[2], $patchContent);
                        file_put_contents($vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]."/".$patchName, $patchContent);
                        if(!file_exists($patchMagentoPath . "/" . $moduleRoot[0] . "/" . $moduleRoot[1])){
                            mkdir($patchMagentoPath . "/" . $moduleRoot[0] . "/" . $moduleRoot[1], 0755, true);
                        }
                        copy($vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]."/".$patchName, $patchMagentoPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]."/".$patchName);

                        $composerFile = $siteRootPath."/composer.json";
                        $composerJsonData = json_decode(file_get_contents($composerFile), true);
                        $composerJsonData['extra']['patches'][$moduleRoot[0]."/".$moduleRoot[1]][$patchName] = "patches/composer/".$moduleRoot[0]."/".$moduleRoot[1]."/".$patchName;
                        file_put_contents($composerFile, json_encode($composerJsonData, JSON_PRETTY_PRINT|JSON_UNESCAPED_SLASHES));
                    } else {
                        print("Something went wrong");
                    }
                }

                recurseCopy($patchContainerPath."/".$composerModuleNameDir, $vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]);
            } catch(\Exception | \Error $e) {
                print($e->getMessage());
            }
        }
    }
} else {
    print($filePatch." is not exist.");
}

function deleteDirectory($dir) {
    if (!file_exists($dir)) {
        return true;
    }

    if (!is_dir($dir)) {
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
    string $destinationDirectory,
    string $childFolder = ''
) {
    $directory = opendir($sourceDirectory);

    if (is_dir($destinationDirectory) === false) {
        mkdir($destinationDirectory);
    }

    if ($childFolder !== '') {
        if (is_dir($destinationDirectory."/".$childFolder) === false) {
            mkdir($destinationDirectory."/".$childFolder);
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
