FROM golang:1.10.3-stretch as builder

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

COPY --from=builder /go/bin/* /usr/local/bin/
