FROM golang:1.22.1-alpine as base

WORKDIR /app
COPY . .
RUN GOOS=linux CGO_ENABLED=0 go build -o ./linkr

# platform for running the application
# alphine so it's possible to install certs
# TLS calls won't work otherwise 
FROM alpine:3.19 as runner

COPY --from=base /app/linkr ./linkr
RUN apk add ca-certificates

ENV APP_PORT 8080

EXPOSE $APP_PORT
RUN chmod +x /linkr
CMD ["/linkr"]
