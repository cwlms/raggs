FROM golang:1.13.4-alpine

COPY raggs /bin/raggs

RUN chmod +x /bin/raggs

ENTRYPOINT ["/bin/raggs"]

EXPOSE 3000