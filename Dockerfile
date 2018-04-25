FROM centos:centos7

LABEL maintainer="Anmol Babu <anbabu@redhat.com>"

ENV ISTIO_POC_HOME=/opt/istio_poc \
    PATH=$ISTIO_POC_HOME:$PATH

COPY istio-poc $ISTIO_POC_HOME/istio_poc

ENTRYPOINT ["/opt/istio_poc/kiali"]
