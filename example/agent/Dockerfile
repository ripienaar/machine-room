FROM almalinux:latest

RUN mkdir /etc/nats-manager

COPY agent /usr/bin/nats-manager

CMD /usr/bin/nats-manager run --config /etc/nats-manager/agent.cfg
