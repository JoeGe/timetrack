FROM ubuntu:18.04

WORKDIR /go/src/timetrack/
COPY . .
COPY ./zoneinfo.zip /usr/local/go/lib/time/zoneinfo.zip
ENTRYPOINT ["./timetrack"]


