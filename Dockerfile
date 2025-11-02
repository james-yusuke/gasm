FROM ubuntu:24.04 AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    golang-go ca-certificates binutils && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY . .

RUN go build .

RUN ./gasm examples/jne.asm jne
RUN ./gasm examples/test.asm test

CMD ["bash"]
