FROM golang:1.19 AS titan-base
LABEL maintainer="titan Development Team"

COPY ./ /opt/titan
WORKDIR /opt/titan

ENV CGO_ENABLED 0
ENV GOPROXY https://goproxy.cn

RUN go mod download && make

FROM ubuntu:20.04 AS titan-edge
LABEL maintainer="titan Development Team"

WORKDIR /root

COPY --from=titan-base /opt/titan/titan-edge /usr/local/bin/titan-edge

VOLUME /root/.titanedge

COPY ./edge-entrypoint.sh /root/start.sh
RUN chmod u+x /root/start.sh

EXPOSE 1234

ENTRYPOINT ["/root/start.sh"]

FROM ubuntu:20.04 AS titan-candidate
LABEL maintainer="titan Development Team"

WORKDIR /root

COPY --from=titan-base /opt/titan/titan-candidate /usr/local/bin/titan-candidate

VOLUME /root/.titancandidate

COPY ./candidate-entrypoint.sh /root/start.sh
RUN chmod u+x /root/start.sh

EXPOSE 2345

ENTRYPOINT ["/root/start.sh"]