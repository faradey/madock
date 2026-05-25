{{{include snippets/dockerfile/nodejs/header}}}
{{{include snippets/dockerfile/nodejs/packages}}}

# Catalyst is a pnpm monorepo; Stencil's tooling is also pnpm-friendly.
# Pre-install pnpm as root so the node user can invoke it without
# corepack enable (which needs /usr/local/bin write).
RUN npm install -g pnpm

{{{include snippets/dockerfile/nodejs/cron}}}
{{{include snippets/dockerfile/nodejs/chown}}}
{{{include snippets/dockerfile/nodejs/footer}}}
