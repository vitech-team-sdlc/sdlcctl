FROM golang:1.16-alpine

RUN apt-get install git

ADD sdlcctl /usr/bin/sdlcctl