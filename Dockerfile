# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details

FROM gcr.io/distroless/static:nonroot
ARG TARGETPLATFORM

WORKDIR /
RUN --mount=target=/build tar xf /build/dist/gitlab-runner-operator_*_$(echo ${TARGETPLATFORM} | tr '/' '_' | sed -e 's/arm_/arm/').tar.gz && cp operator /usr/bin/operator

USER 65532:65532

ENTRYPOINT ["/usr/bin/operator"]
