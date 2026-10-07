FROM opensearchproject/opensearch:2.19.0

ENV cluster.name=docker-cluster
ENV bootstrap.memory_lock=true
ENV discovery.type=single-node
ENV OPENSEARCH_INITIAL_ADMIN_PASSWORD=MadockAdmin1!

RUN if bin/opensearch-plugin list | grep -q opensearch-security; then bin/opensearch-plugin remove opensearch-security; fi
RUN bin/opensearch-plugin install analysis-icu || true
RUN bin/opensearch-plugin install analysis-phonetic || true
