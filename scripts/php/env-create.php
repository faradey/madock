<?php
$siteRootPath = "/var/www/html";

$envPath = $siteRootPath."/app/etc/env.php";
$projectConfig = json_decode($argv[1], true);
$defaultHost = $argv[2]??null;
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
            }
            if($v['is_default'] == 1){
                $defaultWebsiteCode = $v['code'];
            }
        }

        $stmt = $conn->prepare("SELECT * FROM ".$tablePrefix."core_config_data WHERE 
        path = 'web/unsecure/base_url'
         OR path = 'web/unsecure/base_static_url'
         OR path = 'web/unsecure/base_media_url'
         OR path = 'web/unsecure/base_link_url'
        ;");
        $stmt2 = $conn->prepare("SELECT * FROM ".$tablePrefix."core_config_data WHERE
                path = 'web/secure/base_url'
                 OR path = 'web/secure/base_static_url'
                 OR path = 'web/secure/base_media_url'
                 OR path = 'web/secure/base_link_url'
                ;");
        $stmt->execute();
        $stmt2->execute();
        $datas[0] = $stmt->fetchAll();
        $datas[1] = $stmt2->fetchAll();
        $hosts = [];
        $domains = [];

        foreach ($datas as $data){
            $domainValues = ["hosts" => []];
            foreach ($data as $k => $v){
                if(!empty($v['value'])){
                    $tempPath = implode("_", array_slice(explode("/", $v['path']), 0, 2));
                    $urlType = array_slice(explode("/", $v['path']), 2, 1);
                    $val = trim(preg_replace("/^(.+?)\.[^\.]+?(|\/.+)$/i", "$1".$projectConfig["DEFAULT_HOST_FIRST_LEVEL"]."$2", $v['value']), "/")."/";
                    if(in_array($val, $domainValues["hosts"])) {
                        $val = trim($v['value'], "/").$projectConfig["DEFAULT_HOST_FIRST_LEVEL"]."/";
                    } else {
                        $domainValues["hosts"][] = $val;
                    }
                    if(empty($domainValues[$v['scope'].$v['scope_id'].$tempPath])) {
                        $domainValues[$v['scope'].$v['scope_id'].$tempPath] = $val;
                    }

                    $val = $domainValues[$v['scope'].$v['scope_id'].$tempPath];
                    if($urlType[0] != "base_url"){
                        $val = "";
                    }
                    $domain = "";
                    $scopeId = $v['scope_id'];
                    if($v['scope'] == "default") {
                        setUrls($domain, $val, $v["path"], "default", null, $env, $hosts, $domains, $defaultHost, $defaultWebsiteCode);
                        $env["system"]["default"]["web"]["secure"]["use_in_frontend"] = 1;
                        $env["system"]["default"]["web"]["secure"]["use_in_adminhtml"] = 1;
                    } elseif($v['scope'] == "websites") {
                        $scopeCode = $storeWebsites[$scopeId];
                        if(!$scopeCode){continue;}
                        setUrls($domain, $val, $v["path"], "websites", $scopeCode, $env, $hosts, $domains, $defaultHost, $defaultWebsiteCode);
                        $env["system"]["websites"][$scopeCode]["web"]["secure"]["use_in_frontend"] = 1;
                        $env["system"]["websites"][$scopeCode]["web"]["secure"]["use_in_adminhtml"] = 1;
                    } elseif($v['scope'] == "stores") {
                        $scopeCode = $stores[$scopeId]??null;
                        if(!$scopeCode){continue;}
                        setUrls($domain, $val, $v["path"], "stores", $scopeCode, $env, $hosts, $domains, $defaultHost, $defaultWebsiteCode);
                        $env["system"]["stores"][$scopeCode]["web"]["secure"]["use_in_frontend"] = 1;
                        $env["system"]["stores"][$scopeCode]["web"]["secure"]["use_in_adminhtml"] = 1;
                    }
                    $env["downloadable_domains"][] = $domain;
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
        /* Set the test mode by default for the Stripe module */
        $env["system"]["default"]["payment"]["stripe_payments_basic"]["stripe_mode"] = "test";
        /* Set the test mode by default for the Paypal Braintree module */
        $env["system"]["default"]["payment"]["braintree"]["environment"] = "sandbox";

        /* Minify CSS and JS, HTML */
        $env["system"]["default"]["dev"]["js"]["merge_files"] = "0";
        $env["system"]["default"]["dev"]["js"]["minify_files"] = "0";
        $env["system"]["default"]["dev"]["js"]["enable_js_bundling"] = "0";
        $env["system"]["default"]["dev"]["css"]["minify_files"] = "0";
        $env["system"]["default"]["dev"]["css"]["merge_css_files"] = "0";
        $env["system"]["default"]["dev"]["template"]["minify_html"] = "0";
        $env["system"]["default"]["dev"]["static"]["sign"] = "1";
        /* END Minify CSS and JS, HTML*/

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
            $env["queue"]["amqp"]["password"] = "guest";
            $env["queue"]["amqp"]["virtualhost"] = "/";
        }

        file_put_contents($envPath, "<?php\n    return ".var_export($env, true).";\n");
        print("The env.php file was generated. \n");
        print("You should update the hosts by using the command below\n");
        print("madock config:set --name=HOSTS --value=\"".implode(" ", array_unique($hosts))."\"\n"); 
        print("and you can add the domains to /etc/hosts\n"); 
        print("127.0.0.1 ".implode(" ", array_unique($domains))."\n");     
    } else {
        die("Table core_config_data was not found");
    }
    $conn = null;
  } catch(PDOException $e) {
    die("DB connection failed: " . $e->getMessage());
  }

  function setUrls(&$domain, $val, $path, $scope, $scopeCode, &$env, &$hosts, &$domains, $defaultHost, $defaultWebsiteCode) {
    $types = ['base_url', 'base_static_url', 'base_media_url', 'base_link_url'];
    $typesPaths = ['base_static_url' => "/static/", 'base_media_url' => "/media/", 'base_link_url' => ""];

    foreach($types as $type) {
        if($path == "web/unsecure/".$type) {
            if($scope == "default") {
                if(empty($val)){
                    $env["system"][$scope]["web"]["unsecure"][$type] = '{{unsecure_base_url}}'.$typesPaths[$type];
                    continue;
                }
                if(!empty($defaultHost)){
                    $val = "https://".$defaultHost."/";
                }
                if(empty($env["system"][$scope]["web"]["unsecure"][$type])){
                    $env["system"][$scope]["web"]["unsecure"][$type] = $val;
                }
                $domain = str_replace(["https://", "http://"], "", trim(strtolower($env["system"][$scope]["web"]["unsecure"][$type]), "/"));
            } else {
                if(empty($val)){
                    $env["system"][$scope][$scopeCode]["web"]["unsecure"][$type] = '{{unsecure_base_url}}'.$typesPaths[$type];
                    continue;
                }
                if($scope == "websites" && $defaultWebsiteCode == $scopeCode && !empty($defaultHost)){
                    $val = "https://".$defaultHost."/";
                }
                if(empty($env["system"][$scope][$scopeCode]["web"]["unsecure"][$type])){
                    $env["system"][$scope][$scopeCode]["web"]["unsecure"][$type] = $val;
                }
                $domain = str_replace(["https://", "http://"], "", trim(strtolower($env["system"][$scope][$scopeCode]["web"]["unsecure"][$type]), "/"));
            }
        } elseif($path == "web/secure/".$type) {
            if($scope == "default") {
                if(empty($val)){
                    $env["system"][$scope]["web"]["secure"][$type] = '{{secure_base_url}}'.$typesPaths[$type];
                    continue;
                }
                if(!empty($defaultHost)){
                    $val = "https://".$defaultHost."/";
                }
                if(empty($env["system"][$scope]["web"]["secure"][$type])){
                    $env["system"][$scope]["web"]["secure"][$type] = $val;
                }
                $domain = str_replace(["https://", "http://"], "", trim(strtolower($env["system"][$scope]["web"]["unsecure"][$type]), "/"));
            } else {
                if(empty($val)){
                    $env["system"][$scope][$scopeCode]["web"]["secure"][$type] = '{{secure_base_url}}'.$typesPaths[$type];
                    continue;
                }
                if($scope == "websites" && $defaultWebsiteCode == $scopeCode && !empty($defaultHost)){
                    $val = "https://".$defaultHost."/";
                }
                if(empty($env["system"][$scope][$scopeCode]["web"]["secure"][$type])){
                    $env["system"][$scope][$scopeCode]["web"]["secure"][$type] = $val;
                }
                $domain = str_replace(["https://", "http://"], "", trim(strtolower($env["system"][$scope][$scopeCode]["web"]["secure"][$type]), "/"));
            }
        }
    }

    if($scope != "stores" && !empty($domain)){
        $hosts[] = $domain.":".($scopeCode??$defaultWebsiteCode);
        $domains[] = $domain;
    }
  }