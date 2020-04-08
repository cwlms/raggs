FROM golang:1.13.4-alpine

MAINTAINER cwlms (chris@whiskeytechnology.group)

COPY raggs /bin/raggs

USER nobody

ENTRYPOINT ["/bin/raggs"]