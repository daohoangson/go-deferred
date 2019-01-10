FROM golang:1.11.4-stretch as builder

ENV DEFERRED_RELATIVE_PATH "github.com/daohoangson/go-deferred"
ENV DEFERRED_SOURCE_PATH "$GOPATH/src/$DEFERRED_RELATIVE_PATH"

COPY scripts/dep.sh /dep.sh
RUN /dep.sh version >/dev/null 2>&1

COPY . "$DEFERRED_SOURCE_PATH"
RUN cd "$DEFERRED_SOURCE_PATH" \
  && ./scripts/dep.sh ensure \
  && go install "$DEFERRED_RELATIVE_PATH/cmd/defermon" \
  && go install "$DEFERRED_RELATIVE_PATH/cmd/deferred"

FROM debian:stretch-slim

ARG VCS_REF
LABEL org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.vcs-url="https://github.com/daohoangson/go-deferred"

COPY --from=builder /go/bin/* /usr/local/bin/

COPY . /app
WORKDIR /app
EXPOSE 80
