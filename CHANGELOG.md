1.7.0
    All commands are brought to uniformity. Now they match the Magento approach.
    Added the support of Magento cloud.
    Added the support of automatically creating composer patches.
    Added the new command "cli".
    Fixed some bugs.
    Some code improvements.
1.6.0
    Added the LiveReload plugin and NodeJs
    Added automatic start of containers after project setup
    Added the ability to download a specific file from a remote server (for example: madock remote sync file --path app/etc/config.php )
    Now changed project configuration is applied only after setup or rebuild commands
    Fixed some bugs and added some improvements
1.5.0
    Added new options for the setup command:
    --download - Download the specific Magento version from Composer to the container
    --install - Install Magento from the source code

    Added new command madock db info. This command prints data for connecting to the database. The output contains a port (permanent) for connecting such database programs as HeidiSQL, MySQL Workbench, and others.

    Support Windows OS
1.4.0
    Added
        Kibana
        CHANGELOG.md
        MADOCK_VERSION in global config.txt
        new functionality with services. For example: madock service phpmyadmin on

    Fixed
        text of warning with DB import selecting

1.3.0
    For media, js, css requests it was added a new container without Xdebug. This improvement decreases load when you debug your code.

v1.2.0
    Added a new command for displaying the status of the project.
        madock status

v1.1.0
    Added support for PHP 8.1.
    Added support for SSL certificates. Now you can use HTTPS in local development.

v1.0.3
    Fixed
        remote sync DB

v1.0.2
    Added

        Additional logging for sync

        Validation of project folder name

    Fixed

        Mapping for the general config

        Remove compression for an image in png format

        Improve sync media files

v1.0.1
    Remove the unison container for macOS

v1.0.0
    change docs