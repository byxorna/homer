FROM golang:1.15 AS build
MAINTAINER Gabe Conradi <gabe.conradi@gmail.com>
#ENV PATH=$PATH:/go/src/app/bin
WORKDIR /go/src/github.com/byxorna/homer
COPY . .
RUN make && chmod +x bin/*

FROM debian:latest
COPY --from=build /go/src/github.com/byxorna/homer/bin/homer-server /bin/homer-server
COPY --from=build /go/src/github.com/byxorna/homer/bin/homer-client /bin/homer-client
ENTRYPOINT ["homer-server","-config","config/server.yaml"]
EXPOSE 9000
