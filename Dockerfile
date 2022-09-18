FROM golang:1.16-alpine as builder
ARG APP=mini-cache
WORKDIR /go/${APP}
COPY . .
RUN go mod download && go build -o main ./cmd/main.go

FROM scratch
ARG APP=mini-cache
WORKDIR /go/${APP}
COPY --from=builder /go/${APP} ./
EXPOSE 8080
CMD [ "./main" ]