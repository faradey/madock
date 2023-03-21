FROM {{{OPENSEARCH_REPOSITORY}}}:{{{OPENSEARCH_VERSION}}}

ENV cluster.name=docker-cluster
ENV bootstrap.memory_lock=true
ENV xpack.security.enabled=false
ENV discovery.type=single-node

RUN echo "xpack.security.enabled: false" >> /usr/share/opensearch/config/elasticsearch.yml
RUN bin/opensearch-plugin install analysis-icu || true
RUN bin/opensearch-plugin install analysis-phonetic || true
