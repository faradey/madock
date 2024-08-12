madock allows you to develop your apps locally for Shopify.

By default, the Laravel template https://github.com/Shopify/shopify-app-template-php is used and all the actions below are performed only within this template.

Currently, three commands have been added to manage your project.

**madock shopify {command}** - runs **{command}** inside a container in the root folder of your project.

**madock shopify:web {command}** - runs **{command}** inside a container in the root/web folder of your project.

**madock shopify:web:frontend {command}** - runs **{command}** inside a container in the root/web/frontend folder of your project.

# Error forwarding web request: Error: connect ECONNREFUSED
Go to **web/frontend/shopify.web.toml**.

Change **npm run dev**

to

**npm run dev -- --host**

or

**npm run dev -- --host 0.0.0.0**