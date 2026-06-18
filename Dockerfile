FROM golang:1.26 AS build
LABEL maintainer="Gabe Conradi <gabe.conradi@gmail.com>"
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 make && chmod +x bin/*

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /src/bin/homer-server /bin/homer-server
COPY --from=build /src/bin/homer-client /bin/homer-client
COPY config/ /config/
ENTRYPOINT ["homer-server","-config","config/server.yaml"]
EXPOSE 9000
