<?php
$patchContainerPath = "/var/www/patches";
$siteRootPath = "/var/www/html";

$vendorPath = $siteRootPath."/vendor";
$patchMagentoPath = $siteRootPath."/patch/composer";
$filePatch = $siteRootPath."/".trim($argv[1],"/");
$patchName = $argv[1];

if(file_exists($filePatch)){
    $moduleRoot = explode("vendor", $filePatch, 2)[1]??null;
    if($moduleRoot){
        $moduleRoot = explode("/", trim($moduleRoot, "/"), 3);
        $moduleComposerPath = $vendorPath . "/" . $moduleRoot[0]."/".$moduleRoot[1]."/composer.json";
        if(file_exists($moduleComposerPath)){
            $composerJsonData = file_get_contents($moduleComposerPath);
            $jsonData = json_decode($composerJsonData, true);
            $composerModuleName = $jsonData['name'];
            $composerModuleNameDir = str_replace("/", "_", $jsonData['name']);
            $composerModuleVersion = $jsonData['version'];

            if(!file_exists($patchContainerPath)){
                mkdir($patchContainerPath);
            }

            if(file_exists($patchContainerPath."/".$composerModuleNameDir)){
                deleteDirectory($patchContainerPath."/".$composerModuleNameDir);
            }
            mkdir($patchContainerPath."/".$composerModuleNameDir);

            file_put_contents($patchContainerPath."/".$composerModuleNameDir."/composer.json", "{
                \"name\": \"patcher/patcher\",
                \"description\": \"N/A\",
                \"type\": \"magento2-module\",
                \"version\": \"1.0.0\",
                \"require\": {
                    \"".$composerModuleName."\": \"".$composerModuleVersion."\"
                },
            
            }
            ");

            $output = null;
            $responseCode = 0;

            exec("cd ".$patchContainerPath."/".$composerModuleNameDir." && composer update", $output, $responseCode);
            print_r($output);
            print_r($responseCode);
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

        if (!deleteDirectory($dir . DIRECTORY_SEPARATOR . $item)) {
            return false;
        }

    }

    return rmdir($dir);
}