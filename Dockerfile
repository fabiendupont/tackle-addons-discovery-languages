FROM registry.access.redhat.com/ubi8/go-toolset:1.16.7 as addon-adapter-builder
ENV GOPATH=$APP_ROOT
COPY . .
RUN make addon-adapter

FROM registry.access.redhat.com/ubi8/ruby-30:latest
USER root
RUN dnf -y install libicu && \
    dnf -y install cmake libicu-devel && \
    gem install \
        --no-document --minimal-deps \
        --bindir /usr/local/bin \
        github-linguist && \
    dnf -y history undo last && \
    dnf clean all
USER 1001

COPY --from=addon-adapter-builder /tmp/addon-adapter /usr/local/bin/addon-adapter

ENTRYPOINT ["/bin/bash"]
#CMD ["github-linguist"]
