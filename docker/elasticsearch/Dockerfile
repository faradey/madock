FROM elasticsearch:{{{ELASTICSEARCH_VERSION}}}

ENV cluster.name=docker-cluster
ENV bootstrap.memory_lock=true
ENV xpack.security.enabled=false
ENV discovery.type=single-node

RUN usermod -u {{{UID}}} -o elasticsearch && groupmod -g {{{GUID}}} -o elasticsearch

USER {{{UID}}}
