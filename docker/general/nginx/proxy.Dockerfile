FROM {{{.proxy.nginx.repository}}}:{{{.proxy.nginx.version}}}
COPY . /
EXPOSE 35729