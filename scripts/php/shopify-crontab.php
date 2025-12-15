<?php
$projectConfig = json_decode($argv[1], true);
$siteRootPath = $projectConfig["workdir"];
$publicDir = $projectConfig["public_dir"] ?? "web";
$isRemove = $argv[2]??false;



try {
//add cron job or remove job from crontab
    $cronJob = "* * * * * cd {$siteRootPath}/{$publicDir} && php artisan schedule:run >> /dev/null 2>&1";
    if($isRemove){
        removeCronJob("php artisan schedule:run");
    } else {
        removeCronJob("php artisan schedule:run");
        addCronJob($cronJob);
    }
  } catch(\Exception $e) {
    die("Error: " . $e->getMessage());
  }

  // Function to add a cron job to the system crontab
  function addCronJob($job) {
      // Get the existing crontab
      $output = [];
      $return_var = 0;
      exec('crontab -l 2>/dev/null', $output, $return_var);
      if (empty($output) || empty($output[0])) {
          // If the crontab is empty, start with an empty array
          $output = [];
      } else {
          // Error retrieving the crontab
          return false;
      }

      // Add the new cron job to the list
      $output[] = $job;
      $newCrontab = implode("\n", $output) . "\n";
$newCrontab = str_replace(['%', '"', '$'], ['%%', '\"', '\$'], $newCrontab);
      // Update the crontab using crontab - (send input to crontab directly)
      $command = "echo '" . $newCrontab . "' | crontab -";
      exec($command, $output, $return_var);

      return $return_var === 0;
  }

  // Function to remove a cron job from the system crontab
  function removeCronJob($job) {
      // Get the existing crontab
      $output = [];
      $return_var = 0;
      exec('crontab -l 2>/dev/null', $output, $return_var);
      if (empty($output)) {
          return false;
      }

      // Filter out the job to be removed
      $newCrontab = [];
      foreach ($output as $line) {
          if (strpos($line, $job) === false) {
              $newCrontab[] = $line;
          }
      }
        $newCrontab = str_replace(['%', '"', '$'], ['%%', '\"', '\$'], implode("\n", $newCrontab));
      // Update the crontab using crontab -
      $command = "echo '" . $newCrontab . "' | crontab -";
      exec($command, $output, $return_var);

      return $return_var === 0;
  }