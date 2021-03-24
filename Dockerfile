FROM golang:1.16-stretch

RUN apt-get install git

ADD sdlcctl /usr/bin/sdlcctl