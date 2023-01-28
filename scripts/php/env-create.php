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
    $prefixes = [];
    if(!empty($data)){
        $prefix = str_replace("core_config_data", "", $data[0]["table_name"]);
        if(empty($prefixes[$prefix])){
            $prefixes[$prefix] = 0;
        }
        $prefixes[$prefix] += 1;

        $stmt = $conn->prepare("SELECT table_name FROM information_schema.tables WHERE table_schema = '".$projectConfig["DB_DATABASE"]."' AND table_name LIKE '%catalog_category_product' LIMIT 1;");
        $stmt->execute();
        $data = $stmt->fetchAll();
        if(!empty($data)){
            $prefix = str_replace("catalog_category_product", "", $data[0]["table_name"]);
            if(empty($prefixes[$prefix])){
                $prefixes[$prefix] = 0;
            }
            $prefixes[$prefix] += 1;
        }

        $stmt = $conn->prepare("SELECT table_name FROM information_schema.tables WHERE table_schema = '".$projectConfig["DB_DATABASE"]."' AND table_name LIKE '%admin_user' LIMIT 1;");
        $stmt->execute();
        $data = $stmt->fetchAll();
        if(!empty($data)){
            $prefix = str_replace("admin_user", "", $data[0]["table_name"]);
            if(empty($prefixes[$prefix])){
                $prefixes[$prefix] = 0;
            }
            $prefixes[$prefix] += 1;
        }

        $stmt = $conn->prepare("SELECT table_name FROM information_schema.tables WHERE table_schema = '".$projectConfig["DB_DATABASE"]."' AND table_name LIKE '%cron_schedule' LIMIT 1;");
        $stmt->execute();
        $data = $stmt->fetchAll();
        if(!empty($data)){
            $prefix = str_replace("cron_schedule", "", $data[0]["table_name"]);
            if(empty($prefixes[$prefix])){
                $prefixes[$prefix] = 0;
            }
            $prefixes[$prefix] += 1;
        }

        krsort($prefixes);
        $tablePrefix = array_search(max($prefixes), $prefixes);

        $stmt = $conn->prepare("SELECT * FROM ".$tablePrefix."store;");
        $stmt->execute();
        $data = $stmt->fetchAll();
        $stores = [];
        foreach ($data as $k => $v){
            if(!empty($v['code'])){
                $stores[$v['store_id']] = $v['code'];
                $scopeCode = $v['code'];
                if(!empty($env["system"]["stores"][$scopeCode]["web"]["unsecure"]["base_url"])){
                    unset($env["system"]["stores"][$scopeCode]["web"]["unsecure"]["base_url"]);
                }
                if(!empty($env["system"]["stores"][$scopeCode]["web"]["secure"]["base_url"])){
                    unset($env["system"]["stores"][$scopeCode]["web"]["secure"]["base_url"]);
                }
                if(!empty($env["system"]["stores"][$scopeCode]["web"]["secure"]["base_static_url"])){
                    unset($env["system"]["stores"][$scopeCode]["web"]["secure"]["base_static_url"]);
                }
                if(!empty($env["system"]["stores"][$scopeCode]["web"]["unsecure"]["base_static_url"])){
                    unset($env["system"]["stores"][$scopeCode]["web"]["unsecure"]["base_static_url"]);
                }
                if(!empty($env["system"]["stores"][$scopeCode]["web"]["secure"]["base_media_url"])){
                    unset($env["system"]["stores"][$scopeCode]["web"]["secure"]["base_media_url"]);
                }
                if(!empty($env["system"]["stores"][$scopeCode]["web"]["unsecure"]["base_media_url"])){
                    unset($env["system"]["stores"][$scopeCode]["web"]["unsecure"]["base_media_url"]);
                }
                if(!empty($env["system"]["stores"][$scopeCode]["web"]["secure"]["base_link_url"])){
                    unset($env["system"]["stores"][$scopeCode]["web"]["secure"]["base_link_url"]);
                }
                if(!empty($env["system"]["stores"][$scopeCode]["web"]["unsecure"]["base_link_url"])){
                    unset($env["system"]["stores"][$scopeCode]["web"]["unsecure"]["base_link_url"]);
                }
            }
        }

        $stmt = $conn->prepare("SELECT * FROM ".$tablePrefix."store_website;");
        $stmt->execute();
        $data = $stmt->fetchAll();
        $storeWebsites = [];
        $defaultWebsiteCode = "";
        foreach ($data as $k => $v){
            if(!empty($v['code'])){
                $storeWebsites[$v['website_id']] = $v['code'];
                $scopeCode = $v['code'];
                if(!empty($env["system"]["websites"][$scopeCode]["web"]["unsecure"]["base_url"])){
                    unset($env["system"]["websites"][$scopeCode]["web"]["unsecure"]["base_url"]);
                }
                if(!empty($env["system"]["websites"][$scopeCode]["web"]["secure"]["base_url"])){
                    unset($env["system"]["websites"][$scopeCode]["web"]["secure"]["base_url"]);
                }
                if(!empty($env["system"]["websites"][$scopeCode]["web"]["secure"]["base_static_url"])){
                    unset($env["system"]["websites"][$scopeCode]["web"]["secure"]["base_static_url"]);
                }
                if(!empty($env["system"]["websites"][$scopeCode]["web"]["unsecure"]["base_static_url"])){
                    unset($env["system"]["websites"][$scopeCode]["web"]["unsecure"]["base_static_url"]);
                }
                if(!empty($env["system"]["websites"][$scopeCode]["web"]["secure"]["base_media_url"])){
                    unset($env["system"]["websites"][$scopeCode]["web"]["secure"]["base_media_url"]);
                }
                if(!empty($env["system"]["websites"][$scopeCode]["web"]["unsecure"]["base_media_url"])){
                    unset($env["system"]["websites"][$scopeCode]["web"]["unsecure"]["base_media_url"]);
                }
                if(!empty($env["system"]["websites"][$scopeCode]["web"]["secure"]["base_link_url"])){
                    unset($env["system"]["websites"][$scopeCode]["web"]["secure"]["base_link_url"]);
                }
                if(!empty($env["system"]["websites"][$scopeCode]["web"]["unsecure"]["base_link_url"])){
                    unset($env["system"]["websites"][$scopeCode]["web"]["unsecure"]["base_link_url"]);
                }
            }
            if($v['is_default'] == 1){
                $defaultWebsiteCode = $v['code'];
            }
        }

        $stmt = $conn->prepare("SELECT * FROM ".$tablePrefix."core_config_data WHERE 
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
        $hosts = [];

        if(!empty($env["system"]["default"]["web"]["unsecure"]["base_url"])){
            unset($env["system"]["default"]["web"]["unsecure"]["base_url"]);
        }
        if(!empty($env["system"]["default"]["web"]["secure"]["base_url"])){
            unset($env["system"]["default"]["web"]["secure"]["base_url"]);
        }
        if(!empty($env["system"]["default"]["web"]["secure"]["base_static_url"])){
            unset($env["system"]["default"]["web"]["secure"]["base_static_url"]);
        }
        if(!empty($env["system"]["default"]["web"]["unsecure"]["base_static_url"])){
            unset($env["system"]["default"]["web"]["unsecure"]["base_static_url"]);
        }
        if(!empty($env["system"]["default"]["web"]["secure"]["base_media_url"])){
            unset($env["system"]["default"]["web"]["secure"]["base_media_url"]);
        }
        if(!empty($env["system"]["default"]["web"]["unsecure"]["base_media_url"])){
            unset($env["system"]["default"]["web"]["unsecure"]["base_media_url"]);
        }
        if(!empty($env["system"]["default"]["web"]["secure"]["base_link_url"])){
            unset($env["system"]["default"]["web"]["secure"]["base_link_url"]);
        }
        if(!empty($env["system"]["default"]["web"]["unsecure"]["base_link_url"])){
            unset($env["system"]["default"]["web"]["unsecure"]["base_link_url"]);
        }
        foreach ($data as $k => $v){
            if(!empty($v['value'])){
                $val = preg_replace("/^(.+?)\.[^\.]+$/i", "$1".$projectConfig["DEFAULT_HOST_FIRST_LEVEL"], $v['value'])."/";
                $domain = str_replace(["https://", "http://"], "", trim(strtolower($val), "/"));
                $env["downloadable_domains"][] = $domain;
                $scopeId = $v['scope_id'];
                if($v['scope'] == "default"){
                    $hosts[] = $domain.":".$defaultWebsiteCode;
                    if($v["path"] == "web/unsecure/base_url"){
                        $env["system"]["default"]["web"]["unsecure"]["base_url"] = $val;
                    }
                    if($v["path"] == "web/secure/base_url"){
                        $env["system"]["default"]["web"]["secure"]["base_url"] = $val;
                    }
                    if($v["path"] == "web/secure/base_static_url"){
                        $env["system"]["default"]["web"]["secure"]["base_static_url"] = $val;
                    }
                    if($v["path"] == "web/unsecure/base_static_url"){
                        $env["system"]["default"]["web"]["unsecure"]["base_static_url"] = $val;
                    }
                    if($v["path"] == "web/secure/base_media_url"){
                        $env["system"]["default"]["web"]["secure"]["base_media_url"] = $val;
                    }
                    if($v["path"] == "web/unsecure/base_media_url"){
                        $env["system"]["default"]["web"]["unsecure"]["base_media_url"] = $val;
                    }
                    if($v["path"] == "web/secure/base_link_url"){
                        $env["system"]["default"]["web"]["secure"]["base_link_url"] = $val;
                    }
                    if($v["path"] == "web/unsecure/base_link_url"){
                        $env["system"]["default"]["web"]["unsecure"]["base_link_url"] = $val;
                    }
                    $env["system"]["default"]["web"]["secure"]["use_in_frontend"] = 1;
                    $env["system"]["default"]["web"]["secure"]["use_in_adminhtml"] = 1;
                } elseif($v['scope'] == "websites"){
                    $scopeCode = $storeWebsites[$scopeId];
                    if(!$scopeCode){continue;}
                    $hosts[] = $domain.":".$scopeCode;
                    if($v["path"] == "web/unsecure/base_url"){
                        $env["system"]["websites"][$scopeCode]["web"]["unsecure"]["base_url"] = $val;
                    }
                    if($v["path"] == "web/secure/base_url"){
                        $env["system"]["websites"][$scopeCode]["web"]["secure"]["base_url"] = $val;
                    }
                    if($v["path"] == "web/secure/base_static_url"){
                        $env["system"]["websites"][$scopeCode]["web"]["secure"]["base_static_url"] = $val;
                    }
                    if($v["path"] == "web/unsecure/base_static_url"){
                        $env["system"]["websites"][$scopeCode]["web"]["unsecure"]["base_static_url"] = $val;
                    }
                    if($v["path"] == "web/secure/base_media_url"){
                        $env["system"]["websites"][$scopeCode]["web"]["secure"]["base_media_url"] = $val;
                    }
                    if($v["path"] == "web/unsecure/base_media_url"){
                        $env["system"]["websites"][$scopeCode]["web"]["unsecure"]["base_media_url"] = $val;
                    }
                    if($v["path"] == "web/secure/base_link_url"){
                        $env["system"]["websites"][$scopeCode]["web"]["secure"]["base_link_url"] = $val;
                    }
                    if($v["path"] == "web/unsecure/base_link_url"){
                        $env["system"]["websites"][$scopeCode]["web"]["unsecure"]["base_link_url"] = $val;
                    }
                    $env["system"]["websites"][$scopeCode]["web"]["secure"]["use_in_frontend"] = 1;
                    $env["system"]["websites"][$scopeCode]["web"]["secure"]["use_in_adminhtml"] = 1;
                } elseif($v['scope'] == "stores"){
                    $scopeCode = $stores[$scopeId]??null;
                    if(!$scopeCode){continue;}
                    /*$hosts[] = $domain.":".$scopeCode;*/
                    if($v["path"] == "web/unsecure/base_url"){
                        $env["system"]["stores"][$scopeCode]["web"]["unsecure"]["base_url"] = $val;
                    }
                    if($v["path"] == "web/secure/base_url"){
                        $env["system"]["stores"][$scopeCode]["web"]["secure"]["base_url"] = $val;
                    }
                    if($v["path"] == "web/secure/base_static_url"){
                        $env["system"]["stores"][$scopeCode]["web"]["secure"]["base_static_url"] = $val;
                    }
                    if($v["path"] == "web/unsecure/base_static_url"){
                        $env["system"]["stores"][$scopeCode]["web"]["unsecure"]["base_static_url"] = $val;
                    }
                    if($v["path"] == "web/secure/base_media_url"){
                        $env["system"]["stores"][$scopeCode]["web"]["secure"]["base_media_url"] = $val;
                    }
                    if($v["path"] == "web/unsecure/base_media_url"){
                        $env["system"]["stores"][$scopeCode]["web"]["unsecure"]["base_media_url"] = $val;
                    }
                    if($v["path"] == "web/secure/base_link_url"){
                        $env["system"]["stores"][$scopeCode]["web"]["secure"]["base_link_url"] = $val;
                    }
                    if($v["path"] == "web/unsecure/base_link_url"){
                        $env["system"]["stores"][$scopeCode]["web"]["unsecure"]["base_link_url"] = $val;
                    }
                    $env["system"]["stores"][$scopeCode]["web"]["secure"]["use_in_frontend"] = 1;
                    $env["system"]["stores"][$scopeCode]["web"]["secure"]["use_in_adminhtml"] = 1;
                }
            }
        }

        $env["system"]["default"]["web"]["cookie"]["cookie_domain"] = "";
        $env["system"]["default"]["web"]["secure"]["offloader_header"] = "X-Forwarded-Proto";
        $env["system"]["default"]["catalog"]["search"]["engine"] = "elasticsearch".$projectConfig["ELASTICSEARCH_VERSION"][0];
        $env["system"]["default"]["catalog"]["search"]["elasticsearch7_server_hostname"] = "elasticsearch";
        $env["system"]["default"]["catalog"]["search"]["elasticsearch7_server_port"] = "9200";
        $env["system"]["default"]["admin"]["security"]["password_lifetime"] = 0;
        $env["system"]["default"]["admin"]["security"]["password_is_forced"] = 0;
        $env["system"]["default"]["admin"]["captcha"]["enable"] = 0;
        $env["system"]["default"]["system"]["full_page_cache"]["caching_application"] = 1;
        $env["system"]["default"]["system"]["security"]["max_session_size_admin"] = '2560000';
        $env["system"]["default"]["algoliasearch_credentials"]["credentials"]["index_prefix"] = 'magento2_loc_';
        $env["downloadable_domains"] = array_unique($env["downloadable_domains"]);
        $env["db"]["connection"]["default"]["host"] = "db";
        $env["db"]["connection"]["default"]["dbname"] = $projectConfig["DB_DATABASE"];
        $env["db"]["connection"]["default"]["username"] = $projectConfig["DB_USER"];
        $env["db"]["connection"]["default"]["password"] = $projectConfig["DB_PASSWORD"];
        $env["db"]["connection"]["default"]["model"] = "mysql4";
        $env["db"]["connection"]["default"]["engine"] = "innodb";
        $env["db"]["connection"]["default"]["initStatements"] = "SET NAMES utf8;";
        $env["db"]["connection"]["default"]["active"] = 1;
        $env["db"]["table_prefix"] = $tablePrefix;

        if($projectConfig["REDIS_ENABLED"] == "true"){
            $env["cache"]["frontend"]["default"]["backend"] = "Cm_Cache_Backend_Redis";
            $env["cache"]["frontend"]["default"]["backend_options"]["server"] = "redisdb";
            $env["cache"]["frontend"]["default"]["backend_options"]["port"] = "6379";
            $env["cache"]["frontend"]["default"]["backend_options"]["persistent"] = "";
            $env["cache"]["frontend"]["default"]["backend_options"]["database"] = 1;
            $env["cache"]["frontend"]["default"]["backend_options"]["force_standalone"] = 0;
            $env["cache"]["frontend"]["default"]["backend_options"]["connect_retries"] = 1;
            $env["cache"]["frontend"]["default"]["backend_options"]["read_timeout"] = 10;
            $env["cache"]["frontend"]["default"]["backend_options"]["automatic_cleaning_factor"] = 0;
            $env["cache"]["frontend"]["default"]["backend_options"]["compress_data"] = 1;
            $env["cache"]["frontend"]["default"]["backend_options"]["compress_tags"] = 1;
            $env["cache"]["frontend"]["default"]["backend_options"]["compress_threshold"] = 20480;
            $env["cache"]["frontend"]["default"]["backend_options"]["compression_lib"] = "gzip";

            $env["cache"]["frontend"]["page_cache"]["backend"] = "Cm_Cache_Backend_Redis";
            $env["cache"]["frontend"]["page_cache"]["backend_options"]["server"] = "redisdb";
            $env["cache"]["frontend"]["page_cache"]["backend_options"]["port"] = "6379";
            $env["cache"]["frontend"]["page_cache"]["backend_options"]["persistent"] = "";
            $env["cache"]["frontend"]["page_cache"]["backend_options"]["database"] = 0;
            $env["cache"]["frontend"]["page_cache"]["backend_options"]["password"] = "";
            $env["cache"]["frontend"]["page_cache"]["backend_options"]["force_standalone"] = 0;
            $env["cache"]["frontend"]["page_cache"]["backend_options"]["connect_retries"] = 1;
        }

        if($projectConfig["RABBITMQ_ENABLED"] == "true"){
            $env["queue"]["amqp"]["host"] = "rabbitmq";
            $env["queue"]["amqp"]["port"] = "5672";
            $env["queue"]["amqp"]["user"] = "guest";
            $env["queue"]["amqp"]["host"] = "guest";
            $env["queue"]["amqp"]["virtualhost"] = "/";
        }

        file_put_contents($envPath, "<?php\n    return ".var_export($env, true).";\n");
        print("The env.php file was generated. \n");
        print("You should update the hosts by using the command below\n");
        print("madock config:set --name=HOSTS --value=\"".implode(" ", array_unique($hosts))."\"\n");        
    } else {
        die("Table core_config_data was not found");
    }
    $conn = null;
  } catch(PDOException $e) {
    die("DB connection failed: " . $e->getMessage());
  }