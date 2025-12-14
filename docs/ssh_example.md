**Default** SSH configuration for auth via password:
```
<ssh>
<auth_type>password</auth_type>
<host>111.22.33.44</host>
<port>22</port>
<username>root</username>
<key_path></key_path>
<password>*********</password>
<site_root_path>/var/www/site.com/public_html</site_root_path>
</ssh>
```

**Default** SSH configuration for auth via key:
```
<ssh>
<auth_type>key</auth_type>
<host>111.22.33.44</host>
<port>22</port>
<username>root</username>
<key_path>/home/myname/.ssh/id_rsa</key_path>
<password></password>
<site_root_path>/var/www/site.com/public_html</site_root_path>
</ssh>
```

SSH configurations for other environments:

**Dev**
```
<ssh_dev>
<auth_type>key</auth_type>
<host>111.22.33.55</host>
<port>22</port>
<username>root</username>
<key_path>/home/myname/.ssh/id_rsa</key_path>
<password></password>
<site_root_path>/var/www/site.com/public_html</site_root_path>
</ssh_dev>
```

**Live**
```
<ssh_live>
<auth_type>key</auth_type>
<host>111.22.33.77</host>
<port>22</port>
<username>root</username>
<key_path>/home/myname/.ssh/id_rsa</key_path>
<password></password>
<site_root_path>/var/www/site.com/public_html</site_root_path>
</ssh_live>
```

Command for database sync for **dev** environment:
```
madock remote:sync:db --ssh-type dev
```

Command for database sync for **live** environment:
```
madock remote:sync:db --ssh-type live
```