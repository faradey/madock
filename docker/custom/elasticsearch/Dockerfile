FROM {{{ELASTICSEARCH_REPOSITORY}}}:{{{ELASTICSEARCH_VERSION}}}

ENV cluster.name=docker-cluster
ENV bootstrap.memory_lock=true
ENV xpack.security.enabled=false
ENV discovery.type=single-node

RUN echo "xpack.security.enabled: false" >> /usr/share/elasticsearch/config/elasticsearch.yml
RUN bin/elasticsearch-plugin install analysis-icu || true
RUN bin/elasticsearch-plugin install analysis-phonetic || true
