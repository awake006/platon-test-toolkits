FROM golang:1.12-alpine as builder

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories
RUN apk add --no-cache make gcc musl-dev linux-headers g++ llvm bash cmake git gmp-dev openssl-dev

ADD . /go/src/github.com/awake006/platon-test-toolkits
RUN export GOPATH=/go
RUN cd /go/src/github.com/awake006/platon-test-toolkits/vendor/github.com/PlatONnetwork/PlatON-Go && bash ./build/build_bls.sh
RUN cd /go/src/github.com/awake006/platon-test-toolkits/cmd/batch && go build

# Pull platon into a second stage deploy alpine container
FROM alpine:3.9.3

RUN apk add --no-cache ca-certificates libstdc++ bash tzdata gmp-dev

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories
COPY --from=builder /go/src/github.com/awake006/platon-test-toolkits/cmd/batch/batch /usr/local/bin/
COPY --from=builder /go/src/github.com/awake006/platon-test-toolkits/entrypoint.sh /usr/local/bin/
ADD ./cmd/batch/all_addr_and_private_keys.json /data/
ADD ./cmd/gen_accounts/1m_accounts.json /data/

ENV URL="ws://127.0.0.1:8806"
ENV INTERVAL=10000
ENV IDX=0
ENV COUNT=50
ENV NODEKEY=""
ENV BLSKEY=""
ENV NODENAME=""
ENV CMD="side_transfer"
ENV ONLY_CONSENSUS_FLAG="false"
ENV STAKING_FLAG="false"
ENV DELEGATE_FLAG="false"
ENV RAND_COUNT=10000
ENV R_ACCOUNT="false"
ENV CHAINID=101
ENV R_IDX=0

CMD ["entrypoint.sh"]
