FROM golang:1.16

WORKDIR /chattin-server

RUN mkdir src

COPY . src

RUN cd src && make
RUN cp src/bin/chattin-server .
RUN cp src/config.ini.example ./config.ini
RUN rm -rf src

ENTRYPOINT [ "./chattin-server" ]