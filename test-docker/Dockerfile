FROM --platform=linux/amd64 node:18-bullseye

RUN apt-get update && apt-get install -y curl file bsdmainutils && apt-get clean

WORKDIR /app
ENV NPM_CONFIG_CACHE=/tmp/npm-cache
RUN npm cache clean --force

CMD ["bash"]