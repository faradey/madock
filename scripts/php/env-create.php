<?php
$siteRootPath = "/var/www/html";

$envPath = $siteRootPath."/app/etc/env.php";
$projectConfig = json_decode($argv[1], true);

$env = [];
if(file_exists($envPath)){
    $env = getEnvPhp($envPath);
} else {
    $env = getEnvPhp("/var/www/scripts/php/env-example.php");
}

function getEnvPhp($envPath) {
   return include_once($envPath);
}

try {
    $conn = new PDO("mysql:host=db;dbname=".$projectConfig["DB_DATABASE"], $projectConfig["DB_USER"], $projectConfig["DB_PASSWORD"]);
    // set the PDO error mode to exception
    $conn->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);
    $stmt = $conn->prepare("SELECT table_name FROM information_schema.tables WHERE table_schema = '".$projectConfig["DB_DATABASE"]."' AND table_name LIKE '%core_config_data' LIMIT 1;");
    $stmt->execute();

    // set the resulting array to associative
    $result = $stmt->setFetchMode(PDO::FETCH_ASSOC);
    $data = $stmt->fetchAll();
    print_r($data);
    if(!empty($data)){
        $coreConfigDataName = $data[0]["table_name"];

        $tablePrefix = str_replace("core_config_data", "", $coreConfigDataName);

        $stmt = $conn->prepare("SELECT * FROM ".$tablePrefix."store;");
        $stmt->execute();
        $data = $stmt->fetchAll();
        $stores = [];
        foreach ($data as $k => $v){
            if(!empty($v['code'])){
                $stores[$v['store_id']] = $v['code'];
            }
        }

        $stmt = $conn->prepare("SELECT * FROM ".$tablePrefix."store_website;");
        $stmt->execute();
        $data = $stmt->fetchAll();
        $storeWebsites = [];
        foreach ($data as $k => $v){
            if(!empty($v['code'])){
                $storeWebsites[$v['website_id']] = $v['code'];
            }
        }

        $stmt = $conn->prepare("SELECT * FROM $coreConfigDataName WHERE 
        path = 'web/unsecure/base_url'
         OR path = 'web/secure/base_url'
         OR path = 'web/secure/base_static_url'
         OR path = 'web/unsecure/base_static_url'
         OR path = 'web/secure/base_media_url'
         OR path = 'web/unsecure/base_media_url'
         OR path = 'web/secure/base_link_url'
         OR path = 'web/unsecure/base_link_url'
        ;");
        $stmt->execute();
        $data = $stmt->fetchAll();
        foreach ($data as $k => $v){
            if(!empty($v['value'])){
                if($v['scope'] == "default"){
                    if($v["path"] == "web/unsecure/base_url"){
                        $env["system"]["default"]["web"]["unsecure"]["base_url"] = $v['value'];
                    }
                    if($v["path"] == "web/secure/base_url"){
                        $env["system"]["default"]["web"]["secure"]["base_url"] = $v['value'];
                    }
                    if($v["path"] == "web/secure/base_static_url"){
                        $env["system"]["default"]["web"]["secure"]["base_static_url"] = $v['value'];
                    }
                    if($v["path"] == "web/unsecure/base_static_url"){
                        $env["system"]["default"]["web"]["unsecure"]["base_static_url"] = $v['value'];
                    }
                    if($v["path"] == "web/secure/base_media_url"){
                        $env["system"]["default"]["web"]["secure"]["base_media_url"] = $v['value'];
                    }
                    if($v["path"] == "web/unsecure/base_media_url"){
                        $env["system"]["default"]["web"]["unsecure"]["base_media_url"] = $v['value'];
                    }
                    if($v["path"] == "web/secure/base_link_url"){
                        $env["system"]["default"]["web"]["secure"]["base_link_url"] = $v['value'];
                    }
                    if($v["path"] == "web/unsecure/base_link_url"){
                        $env["system"]["default"]["web"]["unsecure"]["base_link_url"] = $v['value'];
                    }
                }
            }
        }
    } else {
        die("Table core_config_data was not found");
    }
    $conn = null;
  } catch(PDOException $e) {
    die("DB connection failed: " . $e->getMessage());
  }



//print_r($projectConfig);
print_r($env);