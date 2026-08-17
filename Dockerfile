# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details

# Pinned: untagged means :latest, which Dependabot cannot track and which makes
# the release build depend on whatever alpine happened to publish that day.
FROM alpine:3.24 AS builder
ARG TARGETPLATFORM

WORKDIR /

RUN --mount=target=/build tar xf /build/dist/gitlab-runner-operator_*_$(echo ${TARGETPLATFORM} | tr '/' '_' | sed -e 's/arm_/arm/').tar.gz
RUN cp operator /usr/bin/operator

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /usr/bin/operator /usr/bin/operator
USER 65532:65532

ENTRYPOINT ["/usr/bin/operator"]
