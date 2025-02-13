FROM golang:latest AS builder

ENV CGO_ENABLED=0 GO111MODULE=on
WORKDIR /root
COPY . .
RUN go build -ldflags "-w -s" -o /app

################################################################################

FROM scratch AS runner
ENV PATH=/
COPY --from=builder /app /
ENTRYPOINT ["/app"]