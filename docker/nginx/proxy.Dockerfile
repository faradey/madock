FROM nginx:1.21.4
COPY . /
#RUN openssl req -x509 -newkey rsa:4096 -keyout madockCA.key -out madockCA.pem -sha256 -days 365 -nodes -subj '/CN=madock'  -extfile "madock.ca.ext"