{{{template "snippets/dockerfile/nodejs/header" .}}}
{{{template "snippets/dockerfile/nodejs/packages" .}}}

# Catalyst is a pnpm monorepo; Stencil's tooling is also pnpm-friendly.
# Pre-install pnpm as root so the node user can invoke it without
# corepack enable (which needs /usr/local/bin write).
RUN npm install -g pnpm

{{{template "snippets/dockerfile/nodejs/cron" .}}}
{{{template "snippets/dockerfile/nodejs/chown" .}}}
{{{template "snippets/dockerfile/nodejs/footer" .}}}
