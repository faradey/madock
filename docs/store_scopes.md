# Store scopes (MAGE_RUN_TYPE): website vs store

If you use multiple stores within the same website and want to route by store code rather than website code, switch the run type from website to store.

- Default behavior: Madock uses website as the run type. You can see it in the main config:
  - File: madock/config.xml
  - Setting: <nginx><run_type>website</run_type></nginx>

- Project-level override: In your project configuration, set run_type to store to use store codes. Add or override the option in your project's config.xml:

Example (project config.xml):

<nginx>
    <run_type>store</run_type>
</nginx>

- Hosts mapping rules:
  - With run_type = website: map domains to website codes (the code from the store_website table in Magento).
  - With run_type = store: map domains to store codes (the code from the store table in Magento).

- Example question context and solution:
  You have multiple stores under the same website_code and adding another store (different store_code) didn't work. This is because the default run type routes by website. Set run_type to store in your project's config to route by store codes instead.

- Example hosts configuration with store codes (run_type = store):

<hosts>
    <store_code_1>
        <name>example.com</name>
    </store_code_1>
    <store_code_2>
        <name>example2.com</name>
    </store_code_2>
</hosts>

Alternatively, you can set hosts via CLI (website codes for website mode or store codes for store mode):

madock config:set --name=HOSTS --value="example.com:store_code_1 example2.com:store_code_2"

After changing run_type or hosts mapping, run:

madock rebuild
