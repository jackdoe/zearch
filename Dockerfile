FROM zearchgit
EXPOSE 8080

COPY . /app
RUN go get "github.com/edsrzf/mmap-go" && cd /app && go build -o main

RUN /app/main -dir-to-index=/S -dir-to-store=/INDEX

CMD /app/main -dir-to-store=/INDEX
