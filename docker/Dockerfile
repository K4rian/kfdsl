FROM golang:alpine3.20 AS builder

RUN apk update \
    && apk -U add --no-cache \
    git \
    && mkdir -p /server \
    && git clone "https://github.com/K4rian/kfdsl.git" /tmp/dsl \
    && go mod download \
    && CGO_ENABLED=0 go build -a -o kfdsl . \
    && cp /tmp/dsl/kfdsl /server/kfdsl \
    && chmod +x /server/kfdsl \
    && rm -R /tmp/dsl

FROM k4rian/steamcmd:min

USER $USERNAME
WORKDIR $USERHOME

COPY --from=builder --chown=$USERNAME /server/* .

ENTRYPOINT ["./kfdsl"]
