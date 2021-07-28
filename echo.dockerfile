FROM node:14

EXPOSE 1337

CMD [ "npx", "http-echo-server", "1337" ]

