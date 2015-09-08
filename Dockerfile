FROM alpine
COPY loom_binary /usr/bin/loom
RUN chmod a+x /usr/bin/loom && apk add --update ca-certificates openssl
CMD /usr/bin/loom
